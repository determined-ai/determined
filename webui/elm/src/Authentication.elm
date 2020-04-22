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


doLogin : Maybe Url.Url -> Cmd msg
doLogin maybeUrl =
    let
        newUrl =
            Route.toString (Route.Login (Maybe.andThen ((\url -> Url.toString url) >> Just) maybeUrl))
    in
    -- load loads all new HTML. It is equivalent to typing the URL into the URL bar and pressing enter.
    -- So whatever is happening in your Model will be thrown out, and a whole new page is loaded.
    -- https://guide.elm-lang.org/webapps/navigation.html
    Navigation.load newUrl


doLogout : Cmd msg
doLogout =
    Route.toString Route.Logout
        |> Navigation.load
