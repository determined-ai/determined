module Page.TrialDetail exposing
    ( Model
    , Msg
    , OutMsg(..)
    , init
    , subscriptions
    , update
    , view
    )

import API
import Browser.Navigation as Navigation
import Communication as Comm
import Components.DropdownSelect as DS
import Components.Logs as Logs
import Components.Table as Table
import Components.Table.Custom as Custom
import Constants
import Dict exposing (Dict)
import Duration
import EverySet
import Formatting
import Html as H exposing (div, text)
import Html.Attributes as HA exposing (class)
import Html.Events as HE
import Html.Lazy
import Json.Decode as D
import Json.Encode as E
import List.Extra
import Maybe.Extra
import Modals.CreateExperiment as MCE
import Modals.ShowCheckpoint as MSCP
import Modules.TensorBoard as TensorBoard
import Page.Common
import Plot
import Round
import Route
import Session exposing (Session)
import Time
import Types
import Utils
import Yaml.Encode



---- Type definitions.


type MetricSpec
    = TrainingMetric String
    | ValidationMetric String


type alias MetricsPoint =
    ( Types.ID, Float )


type Msg
    = StatusTick
    | ToggleShowParams
    | NewStepsTableState Table.State
    | GotTrialDetails Types.TrialDetail
    | GotExperiment Types.ExperimentResult
    | PlotMsg (Plot.Msg MetricsPoint)
    | SelectMetric (Maybe MetricSpec)
    | LogsMsg (Logs.Msg Types.LogEntry)
      -- Continue trial messages.
    | ContinueTrial
    | CreateExpModalMsg MCE.Msg
      -- Checkpoint modals.
    | CheckpointModalMsg MSCP.Msg
    | ShowCheckpoint Types.Checkpoint
      -- TensorBoards. Opening/launching a TensorBoard is a multi-step process
      -- that is routed through GotTensorBoardLaunchCycleMsg.
    | GotTensorBoardLaunchCycleMsg TensorBoard.TensorBoardLaunchCycleMsg
      -- Errors.
    | GotCriticalError Comm.SystemError
    | GotAPIError API.APIError
      -- Filters.
    | ToggleShowHasCheckpoint Bool
    | NewMetricsDropdownState (DS.DropdownState MetricSpec)


type OutMsg
    = NoOp


type alias TrialModel =
    { trial : Types.TrialDetail
    , plotModel : Plot.Model MetricsPoint
    , stepsTableState : Table.State
    , tm : Dict String (List MetricsPoint)
    , vm : Dict String (List MetricsPoint)
    , plottedMetric : Maybe MetricSpec
    , showParams : Bool
    , createExpModalState : MCE.Model
    , checkpointModalState : MSCP.Model
    , logs : Logs.Model Types.LogEntry
    , showLogs : Bool
    , experiment : Maybe Types.Experiment

    -- Filter checkbox state.
    , showHasCheckpoint : Bool
    , metricsDropdownState : DS.DropdownState MetricSpec
    , checkpointModal : Maybe Types.Checkpoint
    }


type ModelState
    = Loading
    | LoadFailed
    | WithTrial TrialModel


type alias Model =
    { id : Types.ID
    , model : ModelState
    }



---- Constants and helpers.


type alias TimedThing a =
    { a
        | startTime : Time.Posix
        , endTime : Maybe Time.Posix
    }


{-| Decides whether a checkpoint has been created or not based on its state.
-}
checkpointWasMade : Types.Checkpoint -> Bool
checkpointWasMade checkpoint =
    case checkpoint.state of
        Types.CheckpointCompleted ->
            True

        Types.CheckpointDeleted ->
            True

        _ ->
            False


totalTime : List (TimedThing a) -> Duration.Duration
totalTime steps =
    List.filter (\step -> Maybe.Extra.unwrap False (always True) step.endTime)
        steps
        |> List.map
            (\step ->
                Duration.from step.startTime (Maybe.withDefault step.startTime step.endTime)
                    |> Duration.inSeconds
            )
        |> List.foldl (+) 0
        |> Duration.seconds


requestHandlers : (body -> Msg) -> API.RequestHandlers Msg body
requestHandlers onSuccess =
    { onSuccess = onSuccess
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIError
    }


{-| For best performance, this element of the logs config must always refer to the same actual
JavaScript object, so define it here rather than inline.
-}
getLogsText : Types.LogEntry -> String
getLogsText =
    .message


logsConfig : Types.ID -> Bool -> Logs.Config Types.LogEntry Msg
logsConfig id keepPolling =
    { toMsg = LogsMsg
    , pollInterval = Logs.defaultPollInterval
    , scrollId = "trial-logs-scroll"
    , containerId = "trial-logs-container"
    , getId = .id
    , getText = getLogsText
    , keepPolling = keepPolling
    , poll = API.pollTrialLogs id
    }


plotConfig : String -> Plot.Config MetricsPoint Msg
plotConfig title =
    { tooltip =
        \( x, y ) ->
            H.span []
                [ H.b [] [ H.text "Step: " ]
                , H.text <| String.fromInt x
                , H.br [] []
                , H.b [] [ H.text "Value: " ]
                , H.text <| String.fromFloat y
                ]
    , toMsg = PlotMsg
    , getX = Tuple.first >> toFloat
    , getY = Tuple.second
    , xLabel = "Step number"
    , yLabel = "Metric value"
    , title = title
    }


batchCol : Int -> Table.Column Types.Step Msg
batchCol batchesPerStep =
    let
        toHtml step =
            case step.checkpoint of
                Just checkpoint ->
                    Page.Common.buttonCreator
                        { action = Page.Common.SendMsg <| ShowCheckpoint checkpoint
                        , bgColor = "blue"
                        , fgColor = "white"
                        , isActive =
                            (checkpoint.state == Types.CheckpointCompleted)
                                || (checkpoint.state == Types.CheckpointDeleted)
                        , isPending = checkpoint.state == Types.CheckpointActive
                        , style = Page.Common.TextOnly
                        , text =
                            checkpoint.stepId
                                * batchesPerStep
                                |> String.fromInt
                        }

                _ ->
                    step.id
                        |> (*) batchesPerStep
                        |> String.fromInt
                        |> H.text

        viewData step =
            { children = [ toHtml step ], attributes = [ class "p-2" ] }

        sorter =
            Table.decreasingOrIncreasingBy (.checkpoint >> Maybe.Extra.unwrap 0 .id)
    in
    Table.veryCustomColumn
        { name = "Batches"
        , id = "batches"
        , sorter = sorter
        , viewData = viewData
        }


metricCol :
    String
    -> List MetricsPoint
    -> Table.Column Types.Step Msg
metricCol metricName metrics =
    let
        maybeMetricValue step =
            List.filter (Tuple.first >> (==) step.id) metrics
                |> List.head
                |> Maybe.map Tuple.second

        metricView step =
            case maybeMetricValue step of
                Just metricValue ->
                    Formatting.validationFormat metricValue |> H.text

                Nothing ->
                    Custom.emptyCell
    in
    Table.veryCustomColumn
        { name = "Metric: " ++ metricName
        , id = "metric"
        , viewData =
            \step ->
                { attributes = [ HA.class "p-2" ], children = [ metricView step ] }

        -- FIXME(hamidzr): We don't know if smaller is better for all the metrics, only the main
        -- validation metric, so for the time being we assume it is.
        , sorter = Custom.maybeNumericalSorter maybeMetricValue True
        }


metricCols : TrialModel -> List MetricSpec -> List (Table.Column Types.Step Msg)
metricCols trialModel metrics =
    List.map
        (\ms ->
            metricCol (getMetricName ms) (getMetricValues trialModel ms)
        )
        metrics


stepsTableConfig : TrialModel -> Table.Config Types.Step Msg
stepsTableConfig trialModel =
    let
        maybeBatchesPerStep =
            Maybe.map (.config >> Utils.batchesPerStep) trialModel.experiment

        selectedMetricCols =
            trialModel.metricsDropdownState.selectedFilters
                |> EverySet.toList
                |> metricCols trialModel
    in
    Table.customConfig
        { toId = .id >> String.fromInt
        , toMsg = NewStepsTableState
        , columns =
            (case maybeBatchesPerStep of
                Just batchesPerStep ->
                    batchCol batchesPerStep

                Nothing ->
                    Table.intColumn "ID" "id" .id
            )
                :: selectedMetricCols
                ++ [ Custom.runStateCol (.state >> Just)
                   ]
        , customizations = Custom.tableCustomizations
        }


getDetailCmd : Types.ID -> Cmd Msg
getDetailCmd id =
    API.pollTrialDetail
        (requestHandlers GotTrialDetails)
        id


getExperimentCmd : TrialModel -> Cmd Msg
getExperimentCmd tm =
    case tm.experiment of
        Just _ ->
            Cmd.none

        Nothing ->
            API.pollExperimentSummary tm.trial.experimentId
                (requestHandlers GotExperiment)


getAllMetricSpecs : Types.TrialDetail -> List MetricSpec
getAllMetricSpecs td =
    let
        ( tm, vm ) =
            getTrialMetrics td

        trainingMetrics =
            Dict.keys tm
                |> List.map TrainingMetric

        valMetrics =
            Dict.keys vm
                |> List.map ValidationMetric
    in
    trainingMetrics ++ valMetrics


autoSelectMetricColumns : List MetricSpec -> Maybe MetricSpec -> EverySet.EverySet MetricSpec
autoSelectMetricColumns metrics plottedMetric =
    if List.length metrics < 4 then
        EverySet.fromList metrics

    else
        Maybe.map List.singleton plottedMetric
            |> Maybe.withDefault []
            |> EverySet.fromList


getBestCheckpoint : Bool -> List Types.Step -> Maybe Types.Checkpoint
getBestCheckpoint smallerIsBetter steps =
    let
        ( selectExtremum, worstMetric ) =
            if smallerIsBetter then
                ( List.Extra.minimumBy, Constants.infinity )

            else
                ( List.Extra.maximumBy, -Constants.infinity )
    in
    steps
        |> List.filter (Utils.getStepValidation >> Maybe.Extra.isJust)
        |> selectExtremum (Utils.getStepValidation >> Maybe.withDefault worstMetric)
        |> Maybe.andThen .checkpoint


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Time.every 5000 (always StatusTick)
        , case model.model of
            WithTrial m ->
                Sub.batch
                    [ MCE.subscriptions m.createExpModalState |> Sub.map CreateExpModalMsg
                    , MSCP.subscriptions |> Sub.map CheckpointModalMsg
                    ]

            _ ->
                Sub.none
        , case model.model of
            WithTrial tm ->
                Logs.subscriptions (logsConfig model.id tm.showLogs) tm.logs
                    |> Sub.map LogsMsg

            _ ->
                Sub.none
        ]



---- Initialization.


init : Types.ID -> ( Model, Cmd Msg )
init id =
    ( { id = id
      , model = Loading
      }
    , getDetailCmd id
    )


initInternal : Types.TrialDetail -> ( TrialModel, Cmd Msg )
initInternal td =
    let
        ( tm, vm ) =
            getTrialMetrics td

        logsDefaultVisibility =
            False

        plottedMetric =
            if Dict.member "loss" tm then
                Just (TrainingMetric "loss")

            else
                Nothing

        ( logsModel, logsCmd ) =
            Logs.init (logsConfig td.id logsDefaultVisibility)

        metrics =
            getAllMetricSpecs td

        defaultMetricDropdownState =
            let
                initialDefault =
                    DS.defaultInitialState metrics
            in
            { initialDefault | selectedFilters = autoSelectMetricColumns metrics plottedMetric }
    in
    ( { trial = td
      , plotModel = Plot.init
      , tm = tm
      , vm = vm
      , plottedMetric = plottedMetric
      , showParams = False
      , stepsTableState = Table.initialSort "ID"
      , createExpModalState = MCE.init
      , checkpointModalState = MSCP.init
      , logs = logsModel
      , showLogs = logsDefaultVisibility
      , checkpointModal = Nothing
      , showHasCheckpoint = True
      , metricsDropdownState = defaultMetricDropdownState
      , experiment = Nothing
      }
    , Cmd.map LogsMsg logsCmd
    )



---- Update.


update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg model session =
    case model.model of
        Loading ->
            case msg of
                GotTrialDetails trial ->
                    let
                        ( m, c ) =
                            initInternal trial
                    in
                    ( { model | model = WithTrial m }
                    , Cmd.batch [ c, getExperimentCmd m ]
                    , Nothing
                    )

                GotCriticalError error ->
                    ( model, Cmd.none, Comm.Error error |> Just )

                GotAPIError e ->
                    let
                        -- TODO(jgevirtz): Report error to user.
                        _ =
                            Debug.log "Got error" e
                    in
                    ( { model | model = LoadFailed }, Cmd.none, Nothing )

                _ ->
                    ( model, Cmd.none, Nothing )

        LoadFailed ->
            case msg of
                StatusTick ->
                    ( model, getDetailCmd model.id, Nothing )

                GotTrialDetails trial ->
                    let
                        ( m, c ) =
                            initInternal trial
                    in
                    ( { model | model = WithTrial m }, c, Nothing )

                GotCriticalError error ->
                    ( model, Cmd.none, Comm.Error error |> Just )

                GotAPIError e ->
                    let
                        -- TODO(jgevirtz): Report error to user.
                        _ =
                            Debug.log "Got error" e
                    in
                    ( model, Cmd.none, Nothing )

                _ ->
                    ( model, Cmd.none, Nothing )

        WithTrial m ->
            let
                ( newModel, cmd, outMsg ) =
                    case msg of
                        StatusTick ->
                            ( m
                            , Cmd.batch
                                [ getDetailCmd model.id
                                , getExperimentCmd m
                                ]
                            , Nothing
                            )

                        ToggleShowParams ->
                            ( { m | showParams = not m.showParams }, Cmd.none, Nothing )

                        NewStepsTableState s ->
                            ( { m | stepsTableState = s }, Cmd.none, Nothing )

                        GotTrialDetails td ->
                            let
                                ( tm, vm ) =
                                    getTrialMetrics td

                                plottedMetric =
                                    if m.plottedMetric == Nothing && Dict.member "loss" tm then
                                        Just (TrainingMetric "loss")

                                    else
                                        m.plottedMetric

                                curDropdownState =
                                    m.metricsDropdownState

                                availableMetricSpecs =
                                    getAllMetricSpecs td

                                neverHadSelectionBefore =
                                    (List.length curDropdownState.options == 0)
                                        && (List.length availableMetricSpecs /= 0)

                                newSelectedFilters =
                                    if neverHadSelectionBefore then
                                        autoSelectMetricColumns availableMetricSpecs plottedMetric

                                    else
                                        curDropdownState.selectedFilters

                                newDropdownState =
                                    { curDropdownState
                                        | options = availableMetricSpecs
                                        , selectedFilters = newSelectedFilters
                                    }
                            in
                            ( { m
                                | trial = td
                                , tm = tm
                                , vm = vm
                                , plottedMetric = plottedMetric
                                , metricsDropdownState = newDropdownState
                              }
                            , Cmd.none
                            , Nothing
                            )

                        PlotMsg pm ->
                            ( { m | plotModel = Plot.update pm m.plotModel }, Cmd.none, Nothing )

                        SelectMetric metric ->
                            ( { m | plottedMetric = metric }, Cmd.none, Nothing )

                        NewMetricsDropdownState state ->
                            ( { m | metricsDropdownState = state }, Cmd.none, Nothing )

                        LogsMsg lm ->
                            let
                                ( logsModel, logsCmd, logsOutMsg ) =
                                    Logs.update (logsConfig model.id m.showLogs) lm m.logs
                            in
                            ( { m | logs = logsModel }
                            , Cmd.map LogsMsg logsCmd
                            , Maybe.map Comm.Error logsOutMsg
                            )

                        GotExperiment (Ok exp) ->
                            ( { m | experiment = Just exp }, Cmd.none, Nothing )

                        GotExperiment (Err _) ->
                            ( m, Cmd.none, Nothing )

                        ContinueTrial ->
                            let
                                ( s, c ) =
                                    MCE.openForContinue m.trial

                                command =
                                    Cmd.map CreateExpModalMsg c
                            in
                            ( { m | createExpModalState = s }, command, Nothing )

                        CreateExpModalMsg subMsg ->
                            let
                                ( newCreateExpModelState, cmd1, outMsg1 ) =
                                    MCE.update subMsg m.createExpModalState

                                ( newModel1, cmd2, outMsg2 ) =
                                    handleCreateExperimentModalMsg outMsg1 m session

                                newModel2 =
                                    { newModel1 | createExpModalState = newCreateExpModelState }

                                commands =
                                    Cmd.batch
                                        [ Cmd.map CreateExpModalMsg cmd1
                                        , cmd2
                                        ]
                            in
                            ( newModel2, commands, outMsg2 )

                        CheckpointModalMsg subMsg ->
                            let
                                ( s, c ) =
                                    MSCP.update subMsg

                                command =
                                    Cmd.map CheckpointModalMsg c
                            in
                            ( { m | checkpointModalState = s }, command, Nothing )

                        ShowCheckpoint checkpoint ->
                            case m.experiment of
                                Just experiment ->
                                    let
                                        ( s, c ) =
                                            MSCP.openCheckpoint experiment checkpoint

                                        command =
                                            Cmd.map CheckpointModalMsg c
                                    in
                                    ( { m | checkpointModalState = s }, command, Nothing )

                                _ ->
                                    ( m, Cmd.none, Nothing )

                        GotTensorBoardLaunchCycleMsg tbLaunchCycle ->
                            let
                                ( c, om ) =
                                    processTensorBoardLaunchCycleMsg tbLaunchCycle
                            in
                            ( m, c, om )

                        GotCriticalError error ->
                            ( m, Cmd.none, Comm.Error error |> Just )

                        GotAPIError e ->
                            let
                                -- TODO(jgevirtz): Report error to user.
                                _ =
                                    Debug.log "Got error" e
                            in
                            ( m, Cmd.none, Nothing )

                        ToggleShowHasCheckpoint isChecked ->
                            ( { m | showHasCheckpoint = isChecked }, Cmd.none, Nothing )
            in
            ( { model | model = WithTrial newModel }, cmd, outMsg )


processTensorBoardLaunchCycleMsg : TensorBoard.TensorBoardLaunchCycleMsg -> ( Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
processTensorBoardLaunchCycleMsg tbLaunchCycleMsg =
    let
        ( reqStatus, cmd ) =
            TensorBoard.handleTensorBoardLaunchCycleMsg
                tbLaunchCycleMsg
                GotTensorBoardLaunchCycleMsg

        outMsg =
            case reqStatus of
                Types.RequestFailed (TensorBoard.Critical e) ->
                    Comm.Error e |> Just

                Types.RequestFailed (TensorBoard.API _) ->
                    -- TODO(jgevirtz): Report error to user.
                    Nothing

                _ ->
                    Nothing
    in
    ( cmd, outMsg )


handleCreateExperimentModalMsg :
    Maybe (Comm.OutMessage MCE.OutMsg)
    -> TrialModel
    -> Session
    -> ( TrialModel, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
handleCreateExperimentModalMsg msg model session =
    case msg of
        Just (Comm.OutMessage (MCE.CreatedExperiment id)) ->
            let
                url =
                    Route.toString (Route.ExperimentDetail id)

                cmd =
                    Navigation.pushUrl session.key url
            in
            ( model, cmd, Nothing )

        Just (Comm.Error e) ->
            ( model, Cmd.none, Comm.Error e |> Just )

        Just (Comm.RaiseToast t) ->
            ( model, Cmd.none, Comm.RaiseToast t |> Just )

        Just (Comm.RouteRequested r w) ->
            ( model, Cmd.none, Comm.RouteRequested r w |> Just )

        Nothing ->
            ( model, Cmd.none, Nothing )


{-| Training metrics values can be null and validation metrics can be null or non-scalar; those
values should be ignored, since we can only show non-null scalar values in the plots. This function
extracts the metrics values from the given trial detail object that are scalars and puts them into a
form that is convenient for later usage (grouped by metric rather than step).
-}
getTrialMetrics :
    Types.TrialDetail
    -> ( Dict String (List MetricsPoint), Dict String (List MetricsPoint) )
getTrialMetrics td =
    let
        -- Take a dict mapping to JSON values and keep only the entries where the value is a scalar.
        filterScalarValues : Dict comparable D.Value -> Dict comparable Float
        filterScalarValues =
            Dict.foldl
                (\k v ->
                    case D.decodeValue D.float v of
                        Ok f ->
                            Dict.insert k f

                        Err _ ->
                            identity
                )
                Dict.empty

        -- Get the ID and metrics from a step, filtering out absent metrics and non-scalar values.
        getScalarStepMetrics :
            (Types.Step -> Maybe (Dict comparable D.Value))
            -> Types.Step
            -> Maybe ( Types.ID, Dict comparable Float )
        getScalarStepMetrics metricsGetter step =
            metricsGetter step |> Maybe.map (\m -> ( step.id, filterScalarValues m ))

        -- Take a list of (ID, metrics dict) pairs extracted from steps and transpose it into a Dict
        -- containing (ID, value) pairs for each named metric.
        accumMetrics : List ( Types.ID, Dict String Float ) -> Dict String (List MetricsPoint)
        accumMetrics =
            List.foldr
                (\( id, stepMetrics ) accum ->
                    Dict.foldl
                        (\k v -> Dict.update k (Maybe.withDefault [] >> (::) ( id, v ) >> Just))
                        accum
                        stepMetrics
                )
                Dict.empty

        validationMetrics =
            td.steps
                |> List.filterMap (getScalarStepMetrics (.validation >> Maybe.andThen .metrics))
                |> accumMetrics

        trainingMetrics =
            td.steps
                |> List.filterMap (getScalarStepMetrics .averageMetrics)
                |> accumMetrics
    in
    ( trainingMetrics, validationMetrics )



---- View.


titleView : TrialModel -> H.Html Msg
titleView model =
    let
        parents =
            [ ( Route.toString <| Route.ExperimentListReact
              , "Experiments"
              )
            , ( Route.toString (Route.ExperimentDetailReact model.trial.experimentId)
              , "Experiment " ++ String.fromInt model.trial.experimentId
              )
            ]
    in
    H.nav [ class "bg-blue-200 p-4" ]
        [ Page.Common.breadcrumb parents (H.text <| "Trial " ++ String.fromInt model.trial.id)
        ]


infoView : TrialModel -> Session -> H.Html Msg
infoView model session =
    let
        mkLabel text maybeHint =
            H.td
                [ class "font-bold p-2 break-words" ]
                (H.text text
                    :: (case maybeHint of
                            Just hint ->
                                [ H.i
                                    [ HA.class "fas fa-question-circle text-blue-600  hover:text-blue-400 ml-1"
                                    , HA.style "cursor" "help"
                                    , HA.title hint
                                    ]
                                    []
                                ]

                            Nothing ->
                                []
                       )
                )

        mkValue value =
            H.td [ class "p-2" ] [ value ]

        stateElem =
            model.trial.state
                |> Page.Common.runStateToSpan

        startTimeElem =
            model.trial.startTime
                |> Formatting.posixToString session.zone
                |> H.text

        endTimeElem =
            model.trial.endTime
                |> Maybe.Extra.unwrap "N/A" (Formatting.posixToString session.zone)
                |> H.text

        paramsElem =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg ToggleShowParams
                , bgColor = "green"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text =
                    if model.showParams then
                        "Hide"

                    else
                        "Show"
                }

        continueTrialButton =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg ContinueTrial
                , bgColor = "blue"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = "Continue Trial"
                }

        logsUrl =
            API.buildUrl [ "det", "trials", String.fromInt model.trial.id, "logs" ] []

        viewLogs =
            Page.Common.buttonCreator
                { action = Page.Common.OpenUrl True logsUrl
                , bgColor = "blue"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = "Logs"
                }

        openTensorBoardButton =
            Page.Common.buttonCreator
                { action =
                    Page.Common.SendMsg
                        (TensorBoard.AccessTensorBoard (Types.FromTrialIds [ model.trial.id ])
                            |> GotTensorBoardLaunchCycleMsg
                        )
                , bgColor = "blue"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = "Open TensorBoard"
                }

        trialCanStillProgress =
            case model.trial.state of
                Types.Active ->
                    True

                _ ->
                    False

        totalDurationView =
            let
                totalDuration =
                    totalTime model.trial.steps

                calcPerSecond count =
                    toFloat count
                        / Duration.inSeconds totalDuration
                        |> Round.round 2

                checkpointDurationView =
                    let
                        checkpoints =
                            List.filterMap .checkpoint model.trial.steps
                                |> List.filter checkpointWasMade
                    in
                    if List.length checkpoints > 0 then
                        let
                            time =
                                totalTime checkpoints
                                    |> Formatting.durationToString
                        in
                        H.li [] [ "Checkpointing: " ++ time |> H.text ]

                    else
                        H.text ""

                validationDurationView =
                    let
                        validations =
                            List.filterMap .validation model.trial.steps
                                |> List.filter (.state >> (==) Types.Completed)
                    in
                    if List.length validations > 0 then
                        let
                            time =
                                totalTime validations
                                    |> Formatting.durationToString
                        in
                        H.li [] [ " Validation: " ++ time |> H.text ]

                    else
                        H.text ""

                maybeBatchesPerStep =
                    Maybe.map (.config >> Utils.batchesPerStep) model.experiment

                completedStepsCount =
                    model.trial.steps
                        |> List.filter (.state >> (==) Types.Completed)
                        |> List.length

                trainingDurationView =
                    if completedStepsCount > 0 then
                        let
                            text =
                                "Training: "
                                    ++ Formatting.durationToString totalDuration
                                    ++ " ("
                                    ++ (case maybeBatchesPerStep of
                                            Just batchesPerStep ->
                                                calcPerSecond (batchesPerStep * completedStepsCount)
                                                    ++ " batches"

                                            Nothing ->
                                                calcPerSecond completedStepsCount
                                                    ++ " steps"
                                       )
                                    ++ "/s)"
                        in
                        H.li [] [ H.text text ]

                    else
                        H.text ""
            in
            if completedStepsCount > 0 then
                H.ul
                    []
                    [ trainingDurationView
                    , validationDurationView
                    , checkpointDurationView
                    ]
                    |> Just

            else if trialCanStillProgress then
                H.text "Waiting for step completion"
                    |> Just

            else
                Nothing

        ( bestValidationElem, bestCheckpointElem ) =
            case model.experiment of
                Just experiment ->
                    let
                        smallerIsBetter =
                            Utils.getSmallerIsBetter experiment.config

                        batchesPerStep =
                            Utils.batchesPerStep experiment.config
                    in
                    case getBestCheckpoint smallerIsBetter model.trial.steps of
                        Just checkpoint ->
                            let
                                buttonLabel =
                                    "Batch " ++ String.fromInt (checkpoint.stepId * batchesPerStep)

                                button =
                                    Page.Common.buttonCreator
                                        { action = Page.Common.SendMsg <| ShowCheckpoint checkpoint
                                        , bgColor = "blue"
                                        , fgColor = "white"
                                        , isActive = True
                                        , isPending = False
                                        , style = Page.Common.TextOnly
                                        , text = buttonLabel
                                        }

                                metricType =
                                    Utils.searcherValidationMetricName experiment.config
                                        |> Maybe.Extra.unwrap "" (\name -> " (" ++ name ++ ")")

                                metricValue =
                                    Maybe.withDefault 0 checkpoint.validationMetric
                                        |> Formatting.validationFormat
                            in
                            ( [ mkLabel ("Best validation" ++ metricType) Nothing
                              , mkValue (H.text metricValue)
                              ]
                            , [ mkLabel "Best checkpoint" Nothing, mkValue button ]
                            )

                        Nothing ->
                            ( [ H.text "" ], [ H.text "" ] )

                Nothing ->
                    ( [ H.text "" ], [ H.text "" ] )

        checkpointsTotalSizeView =
            let
                storedCheckpoints =
                    List.filterMap .checkpoint model.trial.steps
                        |> List.filter (.state >> (==) Types.CheckpointCompleted)

                checkpointSize cp =
                    cp.resources
                        |> Maybe.Extra.unwrap
                            0
                            (Dict.toList >> List.map Tuple.second >> List.foldl (+) 0)

                totalSize =
                    List.map checkpointSize storedCheckpoints
                        |> List.foldl (+) 0
            in
            if totalSize == 0 then
                Nothing

            else
                Just (H.text (Formatting.bytesToString totalSize))

        -- Avoids displaying the row if there is no value.
        situationalRow label maybeHint maybeValue =
            Maybe.Extra.unwrap
                [ H.text "" ]
                (\value -> [ mkLabel label maybeHint, mkValue value ])
                maybeValue

        tableRows =
            [ [ mkLabel "State" Nothing, mkValue stateElem ]
            , [ mkLabel "Start time" Nothing, mkValue startTimeElem ]
            , [ mkLabel "End time" Nothing, mkValue endTimeElem ]
            , [ mkLabel "H-params" (Just "Hyperparameters"), mkValue paramsElem ]
            , situationalRow "Duration" (Just "Wall-clock time of completed work units") totalDurationView
            , bestValidationElem
            , bestCheckpointElem
            , situationalRow "Checkpoint size" (Just "Total stored checkpoint size") checkpointsTotalSizeView
            ]

        table =
            H.table [ class "text-sm text-gray-700 my-2" ] <|
                List.map (H.tr []) tableRows
    in
    div [ class "pb-2 lg:pb-0 border-b lg:border-none lg:pr-4", HA.style "min-width" "15rem" ]
        [ Page.Common.horizontalList
            (continueTrialButton
                :: viewLogs
                :: (if TensorBoard.trialHasMetrics model.trial then
                        [ openTensorBoardButton ]

                    else
                        []
                   )
            )
        , table
        ]


plotsViewHelper : String -> Plot.Model MetricsPoint -> List MetricsPoint -> H.Html Msg
plotsViewHelper =
    Plot.view << plotConfig


plotsView : TrialModel -> H.Html Msg
plotsView model =
    let
        ( title, plotValues ) =
            case model.plottedMetric of
                Just (TrainingMetric s) ->
                    ( "Training metric: " ++ s, Dict.get s model.tm |> Maybe.withDefault [] )

                Just (ValidationMetric s) ->
                    ( "Validation metric: " ++ s, Dict.get s model.vm |> Maybe.withDefault [] )

                Nothing ->
                    ( "", [] )
    in
    div [ class "flex flex-col flex-grow pt-4 lg:pt-0 lg:px-4 lg:border-l", HA.style "min-width" "30rem" ]
        [ div [ class "relative w-full h-full", HA.style "min-height" "300px" ]
            [ Html.Lazy.lazy3 plotsViewHelper
                title
                model.plotModel
                plotValues
            ]
        , -- Only show the metric option if there are any data points; only show the scale option if
          -- there are at least three (with fewer, changing the scale doesn't really do anything
          -- meaningful).
          H.ul [ HA.class "horizontal-list" ]
            [ if not (List.isEmpty plotValues) then
                H.li []
                    [ H.text "Metric: "
                    , metricSelectorView model
                    ]

              else
                H.text ""
            , if List.length plotValues >= 3 then
                H.li []
                    [ H.text "Scale: "
                    , Plot.scaleSelectorView PlotMsg
                    ]

              else
                H.text ""
            ]
        ]


parseMetricSpec : String -> Maybe MetricSpec
parseMetricSpec s =
    if String.startsWith "t-" s then
        Just (TrainingMetric (String.dropLeft 2 s))

    else if String.startsWith "v-" s then
        Just (ValidationMetric (String.dropLeft 2 s))

    else
        Nothing


metricSelectorView : TrialModel -> H.Html Msg
metricSelectorView model =
    let
        optsGroup groupName valuePrefix metricSpec metrics =
            H.optgroup [ HA.attribute "label" groupName ]
                (Dict.keys metrics
                    |> List.map
                        (\name ->
                            let
                                isSelected =
                                    Just (metricSpec name) == model.plottedMetric
                            in
                            H.option
                                [ HA.selected isSelected, HA.value <| valuePrefix ++ name ]
                                [ H.text name ]
                        )
                )
    in
    H.select [ HE.on "change" (D.map (parseMetricSpec >> SelectMetric) (D.at [ "target", "value" ] D.string)) ]
        [ optsGroup "Training metrics" "t-" TrainingMetric model.tm
        , optsGroup "Validation metrics" "v-" ValidationMetric model.vm
        ]


topBoxView : TrialModel -> Session -> H.Html Msg
topBoxView model session =
    div [ class "flex w-full flex-col lg:flex-row text-sm pb-4 text-gray-700" ]
        [ infoView model session, plotsView model ]


paramsView : TrialModel -> H.Html Msg
paramsView model =
    let
        paramsText =
            model.trial.hparams
                |> E.dict identity identity
                |> Yaml.Encode.encode
    in
    if model.showParams then
        Page.Common.section "Hyperparameters" [ H.pre [ HA.class "text-xs" ] [ H.text paramsText ] ]

    else
        H.text ""


filterShowCheckpoint : TrialModel -> Types.Step -> Bool
filterShowCheckpoint tm step =
    Utils.ifThenElse tm.showHasCheckpoint (Maybe.Extra.isJust step.checkpoint) True


filterSteps : TrialModel -> Types.Step -> Bool
filterSteps tm step =
    List.all identity
        [ filterShowCheckpoint tm step
        ]


tableSettings : TrialModel -> H.Html Msg
tableSettings tm =
    let
        showHasCheckpoint =
            H.div []
                [ H.label []
                    [ H.input
                        [ HA.class "mr-1"
                        , HA.type_ "checkbox"
                        , HA.checked tm.showHasCheckpoint
                        , HE.onCheck ToggleShowHasCheckpoint
                        , HA.title "Only show steps that have a checkpoint."
                        ]
                        []
                    , H.text "Has checkpoint"
                    ]
                ]
    in
    H.div [ HA.class "text-gray-700 mb-2" ]
        [ Page.Common.horizontalList
            [ DS.dropDownSelect metricsDropdownConfig tm.metricsDropdownState
            , H.span [ HA.class "border-l py-1 border-gray-700" ] []
            , showHasCheckpoint
            ]
        ]


stepsTableViewHelper : TrialModel -> Table.State -> H.Html Msg
stepsTableViewHelper trialModel tableState =
    let
        table =
            trialModel.trial.steps
                |> List.filter (filterSteps trialModel)
                |> Table.view (stepsTableConfig trialModel) tableState Nothing
    in
    Page.Common.section "Steps"
        [ tableSettings trialModel
        , table
        ]


metricsDropdownConfig : DS.DropdownConfig MetricSpec Msg
metricsDropdownConfig =
    { toMsg = NewMetricsDropdownState
    , orderBySelected = False
    , filtering = False
    , title = "Metrics"
    , filterText = "Displayed metrics"
    , elementToString = getMetricName
    }


getMetricName : MetricSpec -> String
getMetricName spec =
    case spec of
        TrainingMetric metricName ->
            "[Training] " ++ metricName

        ValidationMetric metricName ->
            "[Validation] " ++ metricName


getMetricValues : TrialModel -> MetricSpec -> List MetricsPoint
getMetricValues trialModel spec =
    (case spec of
        TrainingMetric metricName ->
            Dict.get metricName trialModel.tm

        ValidationMetric metricName ->
            Dict.get metricName trialModel.vm
    )
        |> Maybe.withDefault []


stepsTableView : TrialModel -> H.Html Msg
stepsTableView model =
    Html.Lazy.lazy2 stepsTableViewHelper model model.stepsTableState


view : Model -> Session -> H.Html Msg
view model session =
    let
        body =
            case model.model of
                Loading ->
                    [ Page.Common.centeredLoadingWidget ]

                LoadFailed ->
                    [ Page.Common.bigMessage ("Failed to load trial " ++ String.fromInt model.id ++ ".") ]

                WithTrial m ->
                    [ titleView m
                    , div
                        [ class "w-full text-sm p-4" ]
                        [ topBoxView m session
                        , paramsView m
                        , stepsTableView m
                        , MCE.view m.createExpModalState
                            |> H.map CreateExpModalMsg
                        , MSCP.view m.checkpointModalState session
                            |> H.map CheckpointModalMsg
                        ]
                    ]
    in
    div [ class "w-full" ] body
