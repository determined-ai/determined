module Session exposing (Session)

import Browser.Navigation as Navigation
import Time
import Types


type alias Session =
    { key : Navigation.Key
    , zone : Time.Zone
    , user : Maybe Types.SessionUser
    }
