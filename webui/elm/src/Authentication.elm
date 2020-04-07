module Authentication exposing (doLogin, doLogout, getCurrentUser)

import API
import Http
import Json.Decode as Decode
    exposing
        ( Decoder
        , bool
        , string
        , succeed
        )
import Json.Decode.Pipeline as DP exposing (required)
import Json.Encode as E
import Types
import Url.Builder as UB


{-| A structure holding user credentials. Used by Login form.
-}
type alias LoginCredentials =
    { username : String
    , password : String
    }


{-| XHR request to get the currently-authenticated user.
-}
getCurrentUser : (Result Http.Error Types.SessionUser -> m) -> Cmd m
getCurrentUser msg =
    Http.get
        { url = API.buildUrl [ "users", "me" ] []
        , expect = Http.expectJson msg decodeSessionUser
        }


{-| POST credentials to sign in.
-}
doLogin : (Result Http.Error () -> m) -> LoginCredentials -> Cmd m
doLogin msgConstructor credentials =
    Http.post
        { url = API.buildUrl [ "login" ] [ UB.string "cookie" "true" ]
        , body = Http.jsonBody (encodeLoginCredentials credentials)
        , expect = Http.expectWhatever msgConstructor
        }


{-| POST to /logout to log out.
-}
doLogout : (Result Http.Error () -> msg) -> Cmd msg
doLogout tagger =
    Http.post
        { url = API.buildUrl [ "logout" ] []
        , body = Http.emptyBody
        , expect = Http.expectWhatever tagger
        }


encodeLoginCredentials : LoginCredentials -> E.Value
encodeLoginCredentials credentials =
    E.object
        [ ( "username", E.string credentials.username )
        , ( "password", E.string credentials.password )
        ]


{-| Decode a user with extra authenticatication/privilege information.
-}
decodeSessionUser : Decoder Types.SessionUser
decodeSessionUser =
    Decode.succeed Types.SessionUser
        |> DP.custom API.decodeUser
        |> required "admin" bool
        |> required "active" bool
