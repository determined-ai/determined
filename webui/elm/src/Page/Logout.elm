module Page.Logout exposing (Model, Msg, OutMsg(..), init, update, view)

import Authentication
import Communication as Comm
import Html as H
import Html.Attributes as HA
import Http
import Session exposing (Session)


type Status
    = Waiting
    | Done
    | Failed


type alias Model =
    Status


type Msg
    = GotLogout (Result Http.Error ())


type OutMsg
    = LogoutDone


init : ( Model, Cmd Msg )
init =
    ( Waiting, Authentication.doLogout GotLogout )


update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg _ _ =
    case msg of
        GotLogout (Ok ()) ->
            ( Done, Cmd.none, Just (Comm.OutMessage LogoutDone) )

        GotLogout (Err e) ->
            let
                _ =
                    Debug.log "Logout error" e
            in
            ( Failed, Cmd.none, Nothing )


view : Model -> Session -> H.Html Msg
view model _ =
    H.div [ HA.class "container mx-auto" ]
        [ H.div [ HA.class "w-full text-sm p-4" ]
            [ H.text <|
                case model of
                    Waiting ->
                        ""

                    Done ->
                        "Logout finished."

                    Failed ->
                        "Logout failed."
            ]
        ]
