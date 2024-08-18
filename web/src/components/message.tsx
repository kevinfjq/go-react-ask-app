import {ArrowUp} from "lucide-react";
import {useState} from "react";
import {useParams} from "react-router-dom";
import {toast} from "sonner";
import {createMessageReaction} from "../http/create-message-reaction.ts";
import {removeMessageReaction} from "../http/remove-reaction-reaction.ts";

interface MessageProps {
  text: string
  amountOfReactions: number
  answered?: boolean
  id: string
}

export function Message({id: messageId, text, amountOfReactions, answered = false}: MessageProps) {
  const {roomId} = useParams();
  if (!roomId) {
    throw new Error('Messages component must be used within room page');
  }
  const [hasReacted, setHasReacted] = useState(false);

  async function createMessageReactionAction() {
    if (!roomId) {
      return
    }
    try{
      await createMessageReaction({roomId, messageId})
      setHasReacted(true);
    }catch {
      toast.error("Falha ao reagir mensagem, tente novamente")
    }
  }

  async function removeMessageReactionAction() {
    if (!roomId) {
      return
    }
    try{
      await removeMessageReaction({roomId, messageId})
      setHasReacted(false);
    }catch {
      toast.error("Falha ao remover reação mensagem, tente novamente")
    }
  }
  return (
    <li data-answered={answered} className="ml-4 leading-relaxed text-zinc-100 data-[answered=true]:opacity-50 data-[answered=true]:pointer-events-none">
      {text}

      {hasReacted ? (
        <button onClick={removeMessageReactionAction} type="button" className="mt-3 flex items-center gap-2 text-orange-400 text-sm font-medium hover:text-orange-500">
          <ArrowUp className="size-4"/>
          Curtir pergunta ({amountOfReactions})
        </button>
      ) : (
        <button onClick={createMessageReactionAction} type="button" className="mt-3 flex items-center gap-2 text-zinc-400 text-sm font-medium hover:text-zinc-300">
          <ArrowUp className="size-4"/>
          Curtir pergunta ({amountOfReactions})
        </button>
      )}

    </li>


)
}