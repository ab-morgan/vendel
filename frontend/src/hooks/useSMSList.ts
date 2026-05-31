import { queryOptions, useQuery, useSuspenseQuery } from "@tanstack/react-query"
import pb from "@/lib/pocketbase"
import { useRealtimeQuery } from "./useRealtimeQuery"

export type SMSMessageType = "all" | "incoming" | "outgoing"

export const smsListQueryOptions = (messageType: SMSMessageType = "all") =>
  queryOptions({
    queryKey: ["sms", messageType],
    queryFn: async () => {
      const filter =
        messageType === "all" ? "" : `message_type = '${messageType}'`
      const items = await pb.collection("sms_messages").getFullList({
        sort: "-created",
        filter,
        expand: "device",
      })
      return { data: items, count: items.length }
    },
    staleTime: 60_000,
  })

export function useSMSList(messageType: SMSMessageType = "all") {
  useRealtimeQuery("sms_messages", [["sms"], ["quota"]])
  return useQuery(smsListQueryOptions(messageType))
}

export function useSMSListSuspense(messageType: SMSMessageType = "all") {
  useRealtimeQuery("sms_messages", [["sms"], ["quota"]])
  return useSuspenseQuery(smsListQueryOptions(messageType))
}
