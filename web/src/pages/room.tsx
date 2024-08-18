import {useParams} from "react-router-dom";
import amaLogo from '../assets/ama-logo.svg';
import { Share2} from "lucide-react";
import {toast} from "sonner";
import {Messages} from "../components/messages.tsx";
import {Suspense} from "react";
import {CreateMessageForm} from "../components/create-message-form.tsx";



export function Room() {
  const {roomId} = useParams()

  function handleShareRoom() {
    const url = window.location.href.toString()
    if (navigator.share != undefined && navigator.canShare()) {
      navigator.share({url})
    } else {
      navigator.clipboard.writeText(url)
      toast.info("O link da sala foi copiada para a área de transferencia!")
    }
  }
  return (
    <div className="mx-auto max-w-[640px] flex flex-col gap-6 py-10 px-4">
      <div className="flex items-center gap-3 px-3">
        <img src={amaLogo} alt="AMA" className="h-5"/>
        <span className="text-sm text-zinc-500 truncate">Código da sala: <span className="text-zinc-300">{roomId}</span></span>
        <button onClick={handleShareRoom}  type="submit" className="ml-auto bg-zinc-800 text-zinc-300 px-3 py-1.5 gap-1.5 flex items-center rounded-lg font-medium text-sm transition-colors hover:bg-zinc-700">
          Compartilhar
          <Share2 className="size-4"/>
        </button>
      </div>
      <div className="h-px w-full bg-zinc-900"/>
      <CreateMessageForm />

      <Suspense fallback={<p>Carregando...</p>}>
        <Messages />
      </Suspense>
    </div>
  );
}
