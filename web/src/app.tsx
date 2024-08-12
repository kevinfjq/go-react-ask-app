import {createBrowserRouter, RouterProvider} from "react-router-dom";
import {CreateRoom} from "./pages/create-room.tsx";
import {Room} from "./pages/room.tsx";


const router = createBrowserRouter([
  {
    path: '/',
    element: <CreateRoom />
  },
  {
    path: '/room/:roomId',
    element: <Room />
  }
])
export function App() {
  return <RouterProvider router={router} />
}


