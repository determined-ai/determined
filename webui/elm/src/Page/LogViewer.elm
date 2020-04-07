module Page.LogViewer exposing (Model, Msg, OutMsg(..), init, subscriptions, update, view)

import API
import APIQL
import Communication as Comm
import Components.Logs as Logs
import Html as H
import Html.Attributes as HA
import Page.Common
import Ports
import Session exposing (Session)
import Types


type alias Model =
    { logs : Logs.Model Types.LogEntry
    , trialId : Int
    , keepPolling : Bool
    }


type Msg
    = LogsMsg (Logs.Msg Types.LogEntry)


type OutMsg
    = NoOp


{-| Configuration for Components.Logs component.
-}
logsConfig : Types.ID -> Bool -> Logs.Config Types.LogEntry Msg
logsConfig id keepPolling =
    { toMsg = LogsMsg
    , pollInterval = Logs.defaultPollInterval
    , scrollId = "logs-modal-scroll"
    , containerId = "logs-modal-container"
    , getId = .id
    , getText = .message
    , keepPolling = keepPolling
    , poll = \handlers params -> APIQL.sendQuery handlers (APIQL.trialLogsQuery id params)
    }


subscriptions : Model -> Sub Msg
subscriptions m =
    Logs.subscriptions (logsConfig m.trialId m.keepPolling) m.logs
        |> Sub.map LogsMsg


init : Int -> ( Model, Cmd Msg )
init id =
    let
        ( logsModel, logsCmd ) =
            Logs.init <| logsConfig id True
    in
    ( { logs = logsModel
      , trialId = id
      , keepPolling = True
      }
    , Cmd.batch
        [ Cmd.map LogsMsg logsCmd
        , "Trial "
            ++ String.fromInt id
            ++ " Logs - DET"
            |> Ports.setPageTitle
        ]
    )


update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg m _ =
    case msg of
        LogsMsg lm ->
            let
                ( logsModel, logsCmd, logsOutMsg ) =
                    Logs.update (logsConfig m.trialId m.keepPolling) lm m.logs
            in
            ( { m | logs = logsModel }
            , Cmd.map LogsMsg logsCmd
            , Maybe.map Comm.Error logsOutMsg
            )


view : Model -> Session -> H.Html Msg
view m _ =
    H.div [ HA.class "fixed w-screen h-screen top-0 left-0 bg-white" ]
        [ Logs.view (logsConfig m.trialId m.keepPolling)
            m.logs
            [ Page.Common.buttonCreator
                { action = Page.Common.OpenUrl False <| API.trialDetailsPage m.trialId
                , bgColor = "blue"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.IconOnly "icon-experiment"
                , text = "Go back to the trial"
                }
            ]
        ]
