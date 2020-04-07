module Page.Cluster exposing (Model, Msg, OutMsg(..), init, subscriptions, update, view)

import API
import Communication as Comm
import Components.Logs as Logs
import Formatting
import Html as H
import Html.Attributes as HA
import Maybe.Extra
import Page.Common
import Session exposing (Session)
import Time
import Types
import View.SlotChart



---- Constants and helpers.


type ModelState
    = Loading
    | LoadFailed
    | WithCluster (List Types.Slot)


type alias Model =
    { state : ModelState
    , masterLogs : Logs.Model Types.LogEntry
    , showMasterLogs : Bool
    }


type Msg
    = Tick
    | GotSlots (List Types.Slot)
    | LogsMsg (Logs.Msg Types.LogEntry)
    | ToggleMasterLogs
      -- Error handling.
    | GotCriticalError Comm.SystemError
    | GotAPIError API.APIError


type OutMsg
    = SetSlots (List Types.Slot)


requestHandlers : API.RequestHandlers Msg (List Types.Slot)
requestHandlers =
    { onSuccess = GotSlots
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIError
    }


{-| For best performance, this element of the logs config must always refer to the same actual
JavaScript object, so define it here rather than inline.
-}
getLogsText : Types.LogEntry -> String
getLogsText le =
    Maybe.Extra.unwrap "" (\t -> Formatting.posixToString Time.utc t ++ ",") le.time
        ++ Maybe.Extra.unwrap "" (\time -> time ++ ": ") le.level
        ++ le.message


logsConfig : Bool -> Logs.Config Types.LogEntry Msg
logsConfig keepPolling =
    { toMsg = LogsMsg
    , pollInterval = Logs.defaultPollInterval
    , scrollId = "master-logs-scroll"
    , containerId = "master-logs-container"
    , getId = .id
    , getText = getLogsText
    , keepPolling = keepPolling
    , poll = API.pollMasterLogs
    }


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Time.every 2000
            (always Tick)
        , Logs.subscriptions
            (logsConfig model.showMasterLogs)
            model.masterLogs
            |> Sub.map LogsMsg
        ]


init : ( Model, Cmd Msg )
init =
    let
        showMasterLogs =
            False

        ( logsModel, logsCmd ) =
            logsConfig showMasterLogs
                |> Logs.init
    in
    ( { state = Loading, masterLogs = logsModel, showMasterLogs = showMasterLogs }
    , Cmd.batch
        [ API.pollSlots requestHandlers
        , Cmd.map LogsMsg logsCmd
        ]
    )


update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg model _ =
    case msg of
        Tick ->
            ( model, API.pollSlots requestHandlers, Nothing )

        GotSlots slots ->
            ( { model | state = WithCluster slots }
            , Cmd.none
            , Just (Comm.OutMessage (SetSlots slots))
            )

        -- Error handling.
        GotCriticalError error ->
            ( model, Cmd.none, Comm.Error error |> Just )

        GotAPIError e ->
            let
                _ =
                    -- TODO(jgevirtz): Report error to user.
                    Debug.log "Failed to load slots" e
            in
            ( { model | state = LoadFailed }, Cmd.none, Just (Comm.OutMessage (SetSlots [])) )

        LogsMsg lm ->
            let
                ( logsModel, logsCmd, logsOutMsg ) =
                    Logs.update (logsConfig model.showMasterLogs) lm model.masterLogs
            in
            ( { model | masterLogs = logsModel }
            , Cmd.map LogsMsg logsCmd
            , Maybe.map Comm.Error logsOutMsg
            )

        ToggleMasterLogs ->
            let
                ( logsModel, logsCmd, logsOutMsg ) =
                    Logs.update
                        (logsConfig model.showMasterLogs)
                        Logs.DoJumpToBottom
                        model.masterLogs
            in
            ( { model
                | showMasterLogs = not model.showMasterLogs
                , masterLogs = logsModel
              }
            , Cmd.map LogsMsg logsCmd
            , Maybe.map Comm.Error logsOutMsg
            )


slotTypeView : List Types.Slot -> String -> H.Html Msg
slotTypeView slots typeName =
    if List.length slots > 0 then
        Page.Common.unruledSection (typeName ++ " allocation")
            [ H.div [ HA.style "height" "90px" ] [ View.SlotChart.largeView slots ] ]

    else
        H.text (typeName ++ " slots are not available.")


allocationView : List Types.Slot -> H.Html Msg
allocationView slots =
    let
        cpuSlots =
            List.filter (\s -> s.slotType == Types.CPU) slots

        gpuSlots =
            List.filter (\s -> s.slotType == Types.GPU) slots

        cpuCount =
            List.length cpuSlots

        gpuCount =
            List.length gpuSlots

        chartStyle =
            "w-full text-sm text-gray-700 pb-5"

        cpuChart =
            slotTypeView cpuSlots "CPU"

        gpuChart =
            slotTypeView gpuSlots "GPU"

        body =
            case ( cpuCount, gpuCount ) of
                ( _, 0 ) ->
                    [ cpuChart ]

                ( 0, _ ) ->
                    [ gpuChart ]

                _ ->
                    [ cpuChart, gpuChart ]
    in
    H.div [ HA.class chartStyle ] body


agentView : Model -> H.Html Msg
agentView model =
    let
        body =
            case model.state of
                LoadFailed ->
                    Page.Common.bigMessage "Unable to load cluster info."

                WithCluster [] ->
                    Page.Common.bigMessage "No agents connected."

                WithCluster slots ->
                    allocationView slots

                _ ->
                    H.text ""
    in
    body


logsWrapperView : Model -> H.Html Msg -> H.Html Msg
logsWrapperView model logsView =
    H.div [ HA.style "height" "240px" ]
        [ H.hr [] []
        , H.div [ Page.Common.headerClasses ++ "text-2xl" |> HA.class ]
            [ H.text "Master Logs"
            , H.span [ HA.class "text-base" ]
                [ Page.Common.verticalCollapseButton model.showMasterLogs ToggleMasterLogs
                ]
            ]
        , if model.showMasterLogs then
            logsView

          else
            H.text ""
        ]


view : Model -> Session -> H.Html Msg
view model _ =
    let
        body =
            case model.state of
                Loading ->
                    Page.Common.centeredLoadingWidget

                _ ->
                    H.div [ HA.class "w-full text-sm p-4" ]
                        [ Page.Common.pageHeader "Cluster"
                        , agentView model
                        , Logs.view (logsConfig model.showMasterLogs) model.masterLogs []
                            |> logsWrapperView model
                        ]
    in
    H.div [ HA.class "w-full" ] [ body ]
