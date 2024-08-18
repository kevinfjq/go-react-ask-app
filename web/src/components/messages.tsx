import {Message} from "./message.tsx";
import {useParams} from "react-router-dom";
import {getRoomMessages} from "../http/get-room-messages.ts";
import { useSuspenseQuery} from "@tanstack/react-query";
import {useMessagesWebSockets} from "../hooks/use-messages-web-sockets.ts";

export function Messages() {

  const {roomId} = useParams();
  if (!roomId) {
    throw new Error('Messages component must be used within room page');
  }

  const {data} = useSuspenseQuery({
    queryKey: ["messages", roomId],
    queryFn: () => getRoomMessages({roomId}),
  });

  useMessagesWebSockets({roomId})

  const sortedMessages = data.messages.sort((a, b) => {
    return b.amountOfReactions - a.amountOfReactions;
  })

  return (
    <ol className="list-decimal list-outside px-3 space-y-8">
      {sortedMessages.map(message => {
        return (
          <Message text={message.text} amountOfReactions={message.amountOfReactions} answered={message.answered} key={message.id} id={message.id}/>
        )
      })}
    </ol>
  )
}