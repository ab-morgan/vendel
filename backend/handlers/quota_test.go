package handlers

import (
	"net/http"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"vendel/services"
)

// quotaAtLimitApp returns a test app where the user has `remaining` SMS slots
// left in their monthly quota. Passing 0 means the quota is fully exhausted.
func quotaAtLimitApp(t testing.TB, remaining int) *tests.TestApp {
	t.Helper()
	app := setupTestApp(t)

	user, err := app.FindAuthRecordByEmail("users", "user@test.com")
	if err != nil {
		t.Fatal("find test user:", err)
	}

	// Ensure quota record exists (creates it with the free plan on first call).
	if err := services.CreateDefaultQuota(app, user.Id); err != nil {
		t.Fatal("create default quota:", err)
	}

	quota, err := app.FindFirstRecordByFilter(
		"user_quotas", "user = {:uid}", dbx.Params{"uid": user.Id},
	)
	if err != nil {
		t.Fatal("find quota:", err)
	}

	plan, err := app.FindRecordById("user_plans", quota.GetString("plan"))
	if err != nil {
		t.Fatal("find plan:", err)
	}

	limit := plan.GetInt("max_sms_per_month")
	quota.Set("sms_sent_this_month", limit-remaining)
	if err := app.Save(quota); err != nil {
		t.Fatal("save quota:", err)
	}

	return app
}

// testMsgID is a fixed record ID used by smsReportTestApp so scenarios can
// reference it in request bodies without knowing a randomly-generated ID.
const testMsgID = "testmessage0001" // 15 chars — matches PocketBase ID length

// smsReportTestApp returns a test app that has a pre-seeded outgoing message
// (id == testMsgID) assigned to the test device.
func smsReportTestApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app := setupTestApp(t)

	user, err := app.FindAuthRecordByEmail("users", "user@test.com")
	if err != nil {
		t.Fatal("find test user:", err)
	}

	device, err := app.FindFirstRecordByFilter(
		"sms_devices", "user = {:uid}", dbx.Params{"uid": user.Id},
	)
	if err != nil {
		t.Fatal("find test device:", err)
	}

	col, err := app.FindCollectionByNameOrId("sms_messages")
	if err != nil {
		t.Fatal("find sms_messages collection:", err)
	}

	msg := core.NewRecord(col)
	msg.Id = testMsgID
	msg.Set("to", "+1234567890")
	msg.Set("body", "test message")
	msg.Set("user", user.Id)
	msg.Set("device", device.Id)
	msg.Set("status", "assigned")
	msg.Set("message_type", "outgoing")
	msg.Set("webhook_sent", false)
	if err := app.Save(msg); err != nil {
		t.Fatal("save test message:", err)
	}

	return app
}

func TestSMSQuotaEnforcement(t *testing.T) {
	userToken := generateTestToken(t, "users", "user@test.com")

	scenarios := []tests.ApiScenario{
		{
			Name:   "send succeeds when one quota slot remains",
			Method: http.MethodPost,
			URL:    "/api/sms/send",
			Body:   stringBody(`{"recipients":["+1234567890"],"body":"Hello"}`),
			Headers: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"status":"accepted"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return quotaAtLimitApp(t, 1)
			},
		},
		{
			Name:   "send fails with 429 when quota is exhausted",
			Method: http.MethodPost,
			URL:    "/api/sms/send",
			Body:   stringBody(`{"recipients":["+1234567890"],"body":"Hello"}`),
			Headers: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus:  http.StatusTooManyRequests,
			ExpectedContent: []string{`"quota_exceeded"`, `"sms_monthly"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return quotaAtLimitApp(t, 0)
			},
		},
		{
			Name:   "bulk send fails when recipients exceed remaining quota",
			Method: http.MethodPost,
			URL:    "/api/sms/send",
			Body:   stringBody(`{"recipients":["+1111111111","+2222222222","+3333333333"],"body":"Hello"}`),
			Headers: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus:  http.StatusTooManyRequests,
			ExpectedContent: []string{`"quota_exceeded"`, `"sms_monthly"`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return quotaAtLimitApp(t, 2) // 2 remaining, 3 requested
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestSMSReportHappyPath(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:   "device reports message as sent",
			Method: http.MethodPost,
			URL:    "/api/sms/report",
			Body:   stringBody(`{"message_id":"` + testMsgID + `","status":"sent"}`),
			Headers: map[string]string{
				"X-API-Key": "test_device_key_123",
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"success":true`, `"status":"sent"`},
			TestAppFactory:  smsReportTestApp,
		},
		{
			Name:   "device reports message as delivered",
			Method: http.MethodPost,
			URL:    "/api/sms/report",
			Body:   stringBody(`{"message_id":"` + testMsgID + `","status":"delivered"}`),
			Headers: map[string]string{
				"X-API-Key": "test_device_key_123",
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"success":true`, `"status":"delivered"`},
			TestAppFactory:  smsReportTestApp,
		},
		{
			Name:   "device reports message as failed with error",
			Method: http.MethodPost,
			URL:    "/api/sms/report",
			Body:   stringBody(`{"message_id":"` + testMsgID + `","status":"failed","error_message":"No signal"}`),
			Headers: map[string]string{
				"X-API-Key": "test_device_key_123",
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"success":true`, `"status":"failed"`},
			TestAppFactory:  smsReportTestApp,
		},
		{
			Name:   "report for non-existent message returns 404",
			Method: http.MethodPost,
			URL:    "/api/sms/report",
			Body:   stringBody(`{"message_id":"doesnotexist000","status":"sent"}`),
			Headers: map[string]string{
				"X-API-Key": "test_device_key_123",
			},
			ExpectedStatus: http.StatusNotFound,
			TestAppFactory: setupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
