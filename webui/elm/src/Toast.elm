module Toast exposing (Toast, new)

import Process
import Task


type alias Toast =
    { message : String
    , id : Int
    }


timeout : Float
timeout =
    5000.0


new : (Int -> msg) -> Int -> String -> ( Toast, Int, Cmd msg )
new onExpired nextToastID message =
    let
        toast =
            { message = message
            , id = nextToastID
            }

        cmd =
            Task.perform (\() -> onExpired toast.id) (Process.sleep timeout)
    in
    ( toast, nextToastID + 1, cmd )
