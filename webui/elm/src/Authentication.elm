module Authentication exposing
    ( getCurrentUser
    , goToLogin
    , goToLogout
    )

import API
import Http
import Json.Decode as Decode
    exposing
        ( Decoder
        , bool
        , succeed
        )
import Json.Decode.Pipeline as DP exposing (required)
import Ports
import Route
import Types
import Url


{-| XHR request to get the currently-authenticated user.
-}
getCurrentUser : (Result Http.Error Types.SessionUser -> m) -> Cmd m
getCurrentUser msg =
    Http.get
        { url = API.buildUrl [ "users", "me" ] []
        , expect = Http.expectJson msg decodeSessionUser
        }


{-| Decode a user with extra authenticatication/privilege information.
-}
decodeSessionUser : Decoder Types.SessionUser
decodeSessionUser =
    Decode.succeed Types.SessionUser
        |> DP.custom API.decodeUser
        |> required "admin" bool
        |> required "active" bool


goToLogin : Maybe Url.Url -> Cmd msg
goToLogin maybeRedirect =
    Route.Login maybeRedirect
        |> Route.toString
        |> Ports.assignLocation


goToLogout : Cmd msg
goToLogout =
    Route.toString Route.Logout
        |> Ports.assignLocation
