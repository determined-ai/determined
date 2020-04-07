module Modals.Logs exposing
    ( Config
    , Model
    , Msg
    , closed
    , open
    , subscriptions
    , update
    , view
    )

import API
import Communication as Comm
import Components.Logs as Logs
import Formatting
import Html as H
import Html.Attributes as HA
import Types
import View.Modal as Modal


type alias Config =
    { poll :
        API.RequestHandlers (Logs.Msg Types.CommandEvent) (List Types.CommandEvent)
        ->
            { greaterThanId : Maybe Int
            , lessThanId : Maybe Int
            , tailLimit : Maybe Int
            }
        -> Cmd (Logs.Msg Types.CommandEvent)
    , pollInterval : Float
    , description : String
    }


type alias OpenModel =
    Logs.Model Types.CommandEvent


type Model
    = Closed
    | Open Config OpenModel


type Msg
    = CloseModal
    | LogsMsg (Logs.Msg Types.CommandEvent)


{-| commandEventToLogString creates a user-readable message from a CommandEvent.
The logic is borrowed from the CLI code found in cli/command.py::render\_event\_stream.
-}
commandEventToLogString : Types.CommandEvent -> Maybe String
commandEventToLogString event =
    case event.detail of
        Types.ScheduledEvent ->
            "Scheduling "
                ++ event.parentID
                ++ " (id: "
                ++ event.description
                ++ ")..."
                |> Just

        Types.AssignedEvent ->
            event.description
                ++ " was assigned to an agent..."
                |> Just

        Types.ContainerStartedEvent ->
            "Container of "
                ++ event.description
                ++ " has started..."
                |> Just

        Types.TerminateRequestEvent ->
            event.description
                ++ "was requested to terminate..."
                |> Just

        Types.ExitedEvent message ->
            event.description
                ++ " was terminated: "
                ++ message
                |> Just

        Types.LogEvent log ->
            Just log

        Types.ServiceReadyEvent ->
            Nothing


{-| Configuration for Components.Logs component.
-}
logsConfig : Config -> Logs.Config Types.CommandEvent Msg
logsConfig config =
    { toMsg = LogsMsg
    , pollInterval = config.pollInterval
    , scrollId = "logs-modal-scroll"
    , containerId = "logs-modal-container"
    , getId = .seq
    , getText = Formatting.maybeAddNewLine << Maybe.withDefault "" << commandEventToLogString
    , keepPolling = True
    , poll = config.poll
    }


{-| Initialize the Logs modal in the closed state.
-}
closed : Model
closed =
    Closed


{-| Initialize the Logs modal in the open state and return any commands that need to be executed.
-}
open : Config -> ( Model, Cmd Msg )
open config =
    let
        ( logsModel, logsCmd ) =
            Logs.init (logsConfig config)
    in
    ( Open config logsModel
    , Cmd.map LogsMsg logsCmd
    )


{-| Update function for the open state.
-}
updateOpen : Msg -> Config -> OpenModel -> ( Model, Cmd Msg, Maybe Comm.SystemError )
updateOpen msg config m =
    case msg of
        CloseModal ->
            ( Closed, Cmd.none, Nothing )

        LogsMsg lm ->
            let
                ( logsModel, logsCmd, logsOutMsg ) =
                    Logs.update (logsConfig config) lm m
            in
            ( Open config logsModel
            , logsCmd |> Cmd.map LogsMsg
            , logsOutMsg
            )


update : Msg -> Model -> ( Model, Cmd Msg, Maybe Comm.SystemError )
update msg model =
    case model of
        Closed ->
            ( model, Cmd.none, Nothing )

        Open config openModel ->
            updateOpen msg config openModel


subscriptions : Model -> Sub Msg
subscriptions model =
    case model of
        Closed ->
            Sub.none

        Open config openModel ->
            Logs.subscriptions (logsConfig config) openModel
                |> Sub.map LogsMsg


renderModal : H.Html Msg -> H.Html Msg
renderModal c =
    Modal.view
        { content =
            H.div [ HA.class "overflow-y-hidden", HA.style "height" "60vh" ]
                [ H.div [ HA.style "height" "100%", HA.class "relative" ]
                    [ H.div [ HA.class "absolute inset-0" ]
                        [ c ]
                    ]
                ]
        , attributes =
            [ HA.style "width" "60vw"
            ]
        , closeMsg = CloseModal
        }


renderContent : Config -> OpenModel -> H.Html Msg
renderContent c m =
    let
        headerText =
            "Logs for " ++ c.description
    in
    Modal.contentView
        { header =
            H.span
                [ HA.class "text-2xl" ]
                [ H.text headerText ]
        , body =
            H.div
                [ HA.class "h-full pl-2" ]
                [ Logs.view (logsConfig c) m [] ]
        , footer = Nothing
        , buttons = []
        }


view : Model -> H.Html Msg
view model =
    case model of
        Closed ->
            H.text ""

        Open c m ->
            renderContent c m
                |> renderModal
