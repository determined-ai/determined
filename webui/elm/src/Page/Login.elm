module Page.Login exposing (Model, Msg(..), OutMsg(..), init, update, view)

--import Crypto.Hash exposing (sha512)

import Authentication exposing (doLogin, getCurrentUser)
import Binary
import Browser.Navigation as Navigation
import Communication as Comm
import Html as H
import Html.Attributes as HA
import Html.Events as HE
import Http
import Route
import SHA
import Session exposing (Session)
import Types
import Url exposing (Url)


passwordSalt : String
passwordSalt =
    "GubPEmmotfiK9TMD6Zdw"



-- Model


type alias Model =
    { username : String
    , password : String
    , warning : Maybe String
    , loading : Bool
    , redirect : Maybe Url
    }



-- Msg


type Msg
    = AuthComplete (Result Http.Error ())
    | DoLogin
    | UpdateUsername String
    | UpdatePassword String
    | ValidatedAuthentication (Result Http.Error Types.SessionUser)


type OutMsg
    = LoginDone Types.SessionUser


init : ( Model, Cmd Msg )
init =
    ( { username = ""
      , password = ""
      , warning = Nothing
      , loading = False
      , redirect = Nothing
      }
    , Cmd.none
    )



-- Update


{-|

    The flow of messages in this module is similar to flow of messages
    that occurs when the WebUI is first loaded in order to establish
    whether or not a user is currently authenticated.

    Below is a diagram that shows the important parts of the flow.

                              +------------------------+
                              |user submits credentials+------+
                              +-+----------------------+      |
                                |                             |
                                |                             | authentication
                 authentication |                             | fails
                 succeeds       |                             |
                                |                             |
                                |                      +------v------+
                    +-----------v---------+            |display error|
                    |verify authentication|            +-------------+
                  +-+                     +-----+
                  | +---------------------+     |
                  |                             |
    auth is valid |                             | auth invalid
                  |                             |
                  |                             |
        +---------v--------+             +------v------+
        |store session info|             |display error|
        +--------+---------+             +-------------+
                 |
                 |
        +--------v------+
        |route user to  |
        |requested view |
        |               |
        +---------------+

-}
update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg model session =
    case msg of
        UpdateUsername u ->
            ( { model | username = u }, Cmd.none, Nothing )

        UpdatePassword p ->
            ( { model | password = p }, Cmd.none, Nothing )

        DoLogin ->
            handleDoLoginMsg model

        AuthComplete result ->
            handleAuthCompleteMsg model result

        ValidatedAuthentication result ->
            handleValidatedAuthMsg model session result


handleValidatedAuthMsg : Model -> Session -> Result Http.Error Types.SessionUser -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
handleValidatedAuthMsg model session result =
    case result of
        Ok user ->
            let
                newModel =
                    { model | loading = False }
            in
            case model.redirect of
                Just url ->
                    ( newModel, Navigation.pushUrl session.key (Url.toString url), Just (Comm.OutMessage (LoginDone user)) )

                Nothing ->
                    ( newModel, Navigation.load (Route.toString Route.Dashboard), Nothing )

        Err _ ->
            let
                newModel =
                    updateLoginPageWarning (Just "Unknown error occurred while logging in.") model
                        |> updateLoadingOverlay False
            in
            ( newModel, Cmd.none, Nothing )


handleDoLoginMsg : Model -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
handleDoLoginMsg model =
    let
        password =
            saltAndHashPassword model.password

        credentials =
            { username = model.username
            , password = password
            }

        loginAction =
            doLogin AuthComplete credentials

        newModel =
            updateLoadingOverlay True model
    in
    ( newModel, loginAction, Nothing )


httpErrorToString : Http.Error -> String
httpErrorToString error =
    case error of
        Http.BadUrl desc ->
            "bad url: " ++ desc

        Http.Timeout ->
            "timeout"

        Http.NetworkError ->
            "network error"

        Http.BadStatus status ->
            "bad status: " ++ String.fromInt status

        Http.BadBody desc ->
            "bad body: " ++ desc


handleAuthCompleteMsg : Model -> Result Http.Error () -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
handleAuthCompleteMsg model result =
    case result of
        Ok () ->
            let
                action =
                    getCurrentUser ValidatedAuthentication

                newModel =
                    clearPassword model
            in
            ( newModel, action, Nothing )

        Err (Http.BadStatus code) ->
            let
                notification =
                    case code of
                        400 ->
                            Just "Unexpected error: bad request."

                        403 ->
                            Just "Incorrect username/password."

                        x ->
                            Just ("Unknown error with code " ++ String.fromInt x)

                newModel =
                    updateLoginPageWarning notification model
                        |> updateLoadingOverlay False
                        |> clearPassword
            in
            ( newModel, Cmd.none, Nothing )

        Err e ->
            let
                notification =
                    "Unknown error: "
                        ++ httpErrorToString e
                        |> Just

                newModel =
                    updateLoginPageWarning notification model
                        |> updateLoadingOverlay False
                        |> clearPassword
            in
            ( newModel, Cmd.none, Nothing )


updateLoginPageWarning : Maybe String -> Model -> Model
updateLoginPageWarning newWarning model =
    { model | warning = newWarning }


updateLoadingOverlay : Bool -> Model -> Model
updateLoadingOverlay loading model =
    { model | loading = loading }


clearPassword : Model -> Model
clearPassword model =
    { model | password = "" }



-- View


getNotificationHtml : Maybe String -> H.Html Msg
getNotificationHtml maybeWarning =
    case maybeWarning of
        Just message ->
            H.div
                [ HA.class "mb-4 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" ]
                [ H.text message ]

        Nothing ->
            H.text ""


loginContents : Model -> H.Html Msg
loginContents model =
    H.div [ HA.class "flex flex-col justify-center max-w-xs w-full" ]
        [ H.img
            [ HA.class "self-center h-8 mb-4"
            , HA.attribute "style" "width: 256px; height: 40px;"
            , HA.src "/public/images/logo-on-light-horizontal.svg"
            ]
            []
        , getNotificationHtml model.warning
        , H.form
            [ HE.onSubmit DoLogin
            ]
            [ H.label [ HA.class "block text-gray-700 text-sm font-bold mb-2", HA.for "input-username" ] [ H.text "Username" ]
            , H.input
                [ HA.id "input-username"
                , HA.placeholder "Username"
                , HA.value model.username
                , HE.onInput UpdateUsername
                , HA.class "appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline mb-4"
                ]
                []
            , H.label
                [ HA.class "block text-gray-700 text-sm font-bold mb-2", HA.for "input-password" ]
                [ H.text "Password" ]
            , H.input
                [ HA.id "input-password"
                , HA.placeholder "Password"
                , HA.value model.password
                , HE.onInput UpdatePassword
                , HA.type_ "password"
                , HA.class "appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline mb-4"
                ]
                []
            , H.button
                [ HA.class "bg-orange-500 w-full hover:bg-orange-400 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline"
                , HA.type_ "submit"
                ]
                [ H.text "Sign In" ]
            ]
        ]


maybeLoadingOverlay : Bool -> H.Html Msg
maybeLoadingOverlay flag =
    if flag then
        H.div
            [ HA.class "fixed inset-0" ]
            [ H.div [ HA.class "fixed inset-0 w-full h-full opacity-50" ] []
            , H.div [ HA.class "lds-ellipsis center opacity-100" ]
                [ H.div [] []
                , H.div [] []
                , H.div [] []
                ]
            ]

    else
        H.text ""


view : Model -> Session -> H.Html Msg
view model _ =
    H.div [ HA.class "flex items-center justify-center h-full" ]
        [ maybeLoadingOverlay model.loading
        , loginContents model
        ]


saltAndHashPassword : String -> String
saltAndHashPassword p =
    if String.isEmpty p then
        p

    else
        (passwordSalt ++ p)
            |> Binary.fromStringAsUtf8
            |> SHA.sha512
            |> Binary.toHex
            |> String.toLower
            |> Debug.log "password"
