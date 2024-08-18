import {Message} from "./message.tsx";
import {useParams} from "react-router-dom";
import {getRoomMessages} from "../http/get-room-messages.ts";
import { useSuspenseQuery} from "@tanstack/react-query";

export function Messages() {
  const {roomId} = useParams();
  if (!roomId) {
    throw new Error('Messages component must be used within room page');
  }

  const {data} = useSuspenseQuery({
    queryKey: ["messages", roomId],
    queryFn: () => getRoomMessages({roomId}),
  });


  return (
    <ol className="list-decimal list-outside px-3 space-y-8">
      {data.messages.map(message => {
        return (
          <Message text={message.text} amountOfReactions={message.amountOfReactions} answered={message.answered} key={message.id}/>
        )
      })}
    </ol>
  )
}