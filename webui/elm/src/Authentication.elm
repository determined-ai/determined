module Authentication exposing
    ( doLogin
    , doLogout
    , getCurrentUser
    )

import API
import Browser.Navigation as Navigation
import Http
import Json.Decode as Decode
    exposing
        ( Decoder
        , bool
        , succeed
        )
import Json.Decode.Pipeline as DP exposing (required)
import Route
import Session exposing (Session)
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


doLogin : Maybe Url.Url -> Session -> Cmd msg
doLogin maybeUrl session =
    let
        newUrl =
            Route.toString (Route.Login (Maybe.andThen ((\url -> Url.toString url) >> Just) maybeUrl))
    in
    Navigation.pushUrl session.key newUrl


doLogout : Session -> Cmd msg
doLogout session =
    let
        newUrl =
            Route.toString Route.Logout
    in
    Navigation.pushUrl session.key newUrl
