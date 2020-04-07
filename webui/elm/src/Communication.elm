module Communication exposing (OutMessage(..), SystemError(..), map)

import Route


type SystemError
    = AuthenticationError
    | Timeout
    | NetworkDown
    | Unknown


type OutMessage om
    = Error SystemError
    | RaiseToast String
      -- The Bool indicates whether to open the route in a new window.
    | RouteRequested Route.Route Bool
    | OutMessage om


map : (a -> b) -> OutMessage a -> OutMessage b
map fn outMsg =
    case outMsg of
        Error e ->
            Error e

        RaiseToast s ->
            RaiseToast s

        RouteRequested r b ->
            RouteRequested r b

        OutMessage x ->
            fn x |> OutMessage
