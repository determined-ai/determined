module Page.ExperimentDetail exposing
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
import Components.AdvancedButton as Button
import Components.Table as Table
import Components.Table.Custom as Custom
import Constants
import Formatting
import Html as H exposing (Html, div, text)
import Html.Attributes as HA exposing (class)
import Html.Events as HE
import Html.Lazy
import Json.Decode as D
import List.Extra
import Maybe.Extra
import Modals.CreateExperiment as MCE
import Modals.ShowCheckpoint as MSCP
import Modals.ShowConfiguration as MSC
import Modules.TensorBoard as TensorBoard
import Page.Common
import Plot
import Round
import Route
import Session exposing (Session)
import Time exposing (Posix)
import Types
import Utils


plotConfig : Session -> String -> Plot.Config MetricsPoint Msg
plotConfig session title =
    { tooltip =
        \{ trial, elapsedTime, endTime, metric } ->
            H.div []
                [ H.b [] [ H.text "Trial: " ]
                , H.text <| String.fromInt trial
                , H.br [] []
                , H.b [] [ H.text "Time: " ]
                , H.text <| Formatting.posixToString session.zone endTime
                , H.br [] []
                , H.b [] [ H.text "Elapsed time: " ]
                , H.text <| Round.round 0 elapsedTime ++ " seconds"
                , H.br [] []
                , H.b [] [ H.text "Value: " ]
                , H.text <| Formatting.validationFormat metric
                ]
    , toMsg = PlotMsg
    , getX = .elapsedTime
    , getY = .metric
    , xLabel = "Elapsed time (seconds)"
    , yLabel = "Metric value"
    , title = title
    }


validationCol :
    String
    -> String
    -> (Types.TrialSummary -> Maybe Float)
    -> Types.Experiment
    -> Table.Column Types.TrialSummary Msg
validationCol name id toMaybeFloat experiment =
    Table.veryCustomColumn
        { name = name
        , id = id
        , viewData =
            \trial ->
                let
                    maybeFloatValue =
                        toMaybeFloat trial

                    title =
                        Maybe.Extra.unwrap [] (\t -> [ HA.title (String.fromFloat t) ]) maybeFloatValue

                    displayNumber =
                        text (Maybe.Extra.unwrap "" Formatting.validationFormat maybeFloatValue)
                in
                { attributes =
                    class "p-2" :: title
                , children =
                    [ displayNumber ]
                }
        , sorter = Custom.maybeNumericalSorter toMaybeFloat (Utils.getSmallerIsBetter experiment.config)
        }


checkpointCol : Int -> Table.Column Types.TrialSummary Msg
checkpointCol batchesPerStep =
    let
        openCheckpointBtn checkpoint =
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
                    "Batch " ++ String.fromInt (checkpoint.stepId * batchesPerStep)
                }

        toHtml trial =
            Maybe.Extra.unwrap
                Custom.emptyCell
                openCheckpointBtn
                trial.bestAvailableCheckpoint

        viewData trial =
            { children = [ toHtml trial ], attributes = [ class "p-2" ] }

        sorter =
            Table.decreasingOrIncreasingBy (.bestAvailableCheckpoint >> Maybe.Extra.unwrap 0 .id)
    in
    Table.veryCustomColumn
        { name = "Checkpoint"
        , id = "checkpoint"
        , sorter = sorter
        , viewData = viewData
        }


trialsTableConfig : Session -> ExperimentModel -> Table.Config Types.TrialSummary Msg
trialsTableConfig session m =
    let
        actionClasses =
            HA.class "font-semibold text-blue-500 hover:text-blue-300 cursor-pointer"

        nTrials =
            m.experiment.trials
                |> Maybe.withDefault []
                |> List.length

        batchesPerStep =
            Utils.batchesPerStep m.experiment.config

        showMore =
            if nTrials > m.numTrialsToShow then
                let
                    remainder =
                        min trialShowIncrement (nTrials - m.numTrialsToShow)
                in
                Just
                    (H.span
                        [ actionClasses, HE.onClick (ShowMore remainder) ]
                        [ "Show " ++ String.fromInt remainder ++ " more" |> H.text ]
                    )

            else
                Nothing

        showLess =
            if m.numTrialsToShow > trialShowIncrement then
                let
                    reduceCount =
                        let
                            remainder =
                                modBy trialShowIncrement m.numTrialsToShow
                        in
                        if remainder /= 0 then
                            remainder

                        else
                            trialShowIncrement
                in
                Just
                    (H.span
                        [ actionClasses, HE.onClick (ShowLess reduceCount) ]
                        [ "Show " ++ String.fromInt reduceCount ++ " fewer" |> H.text ]
                    )

            else
                Nothing

        showAll =
            if nTrials > m.numTrialsToShow then
                Just
                    (H.span
                        [ actionClasses, HE.onClick ShowAll ]
                        [ "Show all (" ++ String.fromInt nTrials ++ ")" |> H.text ]
                    )

            else
                Nothing

        divider =
            H.div [ HA.class "inline-block border-l border-black mx-4" ] []

        showList =
            [ showMore, showLess, showAll ]
                |> Maybe.Extra.values
                |> List.intersperse divider

        footer =
            { attributes = []
            , children =
                [ H.tr []
                    [ H.td [ HA.colspan 100 ]
                        [ H.div [ HA.class "flex flex-row justify-center" ] showList ]
                    ]
                ]
            }
    in
    Table.customConfig
        { toId = .id >> String.fromInt
        , toMsg = NewTrialsTableState
        , columns =
            [ Table.veryCustomColumn
                { name = "ID"
                , id = "id"
                , viewData =
                    \trial ->
                        { attributes = [ HA.class "p-2" ]
                        , children =
                            [ H.a
                                [ HA.href <| Route.toString (Route.TrialDetail trial.id)
                                , Page.Common.onClickStopPropagation NoOp
                                ]
                                [ H.text <| String.fromInt trial.id ]
                            ]
                        }
                , sorter = Table.increasingOrDecreasingBy .id
                }
            , Custom.runStateCol (.state >> Just)
            , Custom.intCol "Steps" "steps" .numSteps
            , validationCol "Best validation" "best-validation" .bestValidationMetric m.experiment
            , validationCol "Latest validation" "latest-validation" .latestValidationMetric m.experiment
            , checkpointCol batchesPerStep
            , Custom.datetimeCol session.zone "Start Time" "start-time" (.startTime >> Time.posixToMillis)
            , Custom.datetimeCol session.zone "End Time" "end-time" (.endTime >> Maybe.Extra.unwrap -1 Time.posixToMillis)
            ]
        , customizations =
            let
                baseCustomizations =
                    Custom.tableCustomizations

                withFooter =
                    { baseCustomizations | tfoot = Just footer }

                rowAttrs trial =
                    [ HA.class "cursor-pointer hover:bg-orange-100"
                    , HE.on "click"
                        (D.map (SendOut << Comm.RouteRequested (Route.TrialDetail trial.id))
                            (D.map2 (||) (D.field "ctrlKey" D.bool) (D.field "metaKey" D.bool))
                        )
                    ]
            in
            { withFooter | rowAttrs = rowAttrs }
        }


type alias MetricsPoint =
    { trial : Types.ID
    , endTime : Posix
    , elapsedTime : Float
    , metric : Float
    }


type alias Model =
    { id : Types.ID
    , model : ModelState
    }


type ModelState
    = Loading
    | LoadFailed
    | WithExperiment ExperimentModel


type alias ExperimentModel =
    { experiment : Types.Experiment
    , plotModel : Plot.Model MetricsPoint
    , trialsTableState : Table.State
    , pendingAction : Maybe Action
    , checkpointModalState : MSCP.Model
    , configModalState : MSC.Model
    , createExpModalState : MCE.Model
    , numTrialsToShow : Int
    , killBtn : Button.Model Msg
    , cancelBtn : Button.Model Msg
    }


type ButtonID
    = KillBtn
    | CancelBtn


getBtnModel : ExperimentModel -> ButtonID -> Button.Model Msg
getBtnModel model id =
    case id of
        KillBtn ->
            model.killBtn

        CancelBtn ->
            model.cancelBtn


updateBtnModel : ExperimentModel -> ButtonID -> Button.Model Msg -> ExperimentModel
updateBtnModel model id btnModel =
    case id of
        KillBtn ->
            { model | killBtn = btnModel }

        CancelBtn ->
            { model | cancelBtn = btnModel }


trialShowIncrement : Int
trialShowIncrement =
    10


getSummaryCmd : Maybe Action -> Int -> Cmd Msg
getSummaryCmd action id =
    requestHandlers action (GotExperiment action)
        |> API.pollExperimentSummary id


getBestCheckpoint : Types.Experiment -> Maybe Types.Checkpoint
getBestCheckpoint experiment =
    let
        ( selectExtremum, worstMetric ) =
            if Utils.getSmallerIsBetter experiment.config then
                ( List.Extra.minimumBy, Constants.infinity )

            else
                ( List.Extra.maximumBy, -Constants.infinity )
    in
    experiment.trials
        |> Maybe.withDefault []
        -- Only deal with trials with checkpoints and validation metrics.
        |> List.filterMap .bestAvailableCheckpoint
        |> selectExtremum (.validationMetric >> Maybe.withDefault worstMetric)


init : Int -> ( Model, Cmd Msg )
init id =
    ( { id = id
      , model = Loading
      }
    , getSummaryCmd Nothing id
    )


initInternal : Types.Experiment -> ExperimentModel
initInternal exp =
    let
        btnModel text confirmBtnText action =
            { needsConfirmation = True
            , confirmBtnText = confirmBtnText
            , dismissBtnText = "Dismiss"
            , promptOpen = False
            , promptTitle = "Confirm Action"
            , promptText = "Are you sure?"
            , config =
                { action = Page.Common.SendMsg (Do action)
                , bgColor = "orange"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = text
                }
            }
    in
    { experiment = exp
    , plotModel = Plot.init
    , trialsTableState = Table.initialSort "best-validation"
    , pendingAction = Nothing
    , checkpointModalState = MSCP.init
    , configModalState = MSC.init
    , createExpModalState = MCE.init
    , numTrialsToShow = trialShowIncrement
    , killBtn = btnModel "Kill" "Kill Experiment" Kill
    , cancelBtn = btnModel "Kill" "Cancel Experiment" Cancel
    }


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Time.every 5000 (always Tick)
        , case model.model of
            WithExperiment expModel ->
                Sub.batch
                    [ MSCP.subscriptions |> Sub.map CheckpointModalMsg
                    , MSC.subscriptions |> Sub.map ConfigModalMsg
                    , MCE.subscriptions expModel.createExpModalState
                        |> Sub.map CreateExpModalMsg
                    ]

            _ ->
                Sub.none
        ]


type Msg
    = NoOp
    | Tick
    | GotExperiment (Maybe Action) Types.ExperimentResult
    | PlotMsg (Plot.Msg MetricsPoint)
    | ShowCheckpoint Types.Checkpoint
    | ShowConfig
    | NewTrialsTableState Table.State
    | SendOut (Comm.OutMessage OutMsg)
    | Do Action
    | Done Action ()
      -- Fork experiment messages.
    | ForkExperiment
    | CheckpointModalMsg MSCP.Msg
    | ConfigModalMsg MSC.Msg
    | CreateExpModalMsg MCE.Msg
      -- TensorBoards. Opening/launching a TensorBoard is a multi-step process
      -- that is routed through GotTensorBoardLaunchCycleMsg.
    | GotTensorBoardLaunchCycleMsg TensorBoard.TensorBoardLaunchCycleMsg
      -- Trial table control.
    | ShowLess Int
    | ShowMore Int
    | ShowAll
      -- Error handling.
    | GotCriticalError Comm.SystemError
    | GotAPIError (Maybe Action) API.APIError
      -- Button
    | ButtonMsg ButtonID Button.Msg


type OutMsg
    = AuthenticationFailure


type Action
    = Activate
    | Archive Bool
    | Cancel
    | Kill
    | Pause


requestHandlers : Maybe Action -> (body -> Msg) -> API.RequestHandlers Msg body
requestHandlers maybeAction onSuccess =
    { onSuccess = onSuccess
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIError maybeAction
    }


actionCommand : Action -> Types.ID -> Cmd Msg
actionCommand action =
    let
        handlers =
            requestHandlers (Just action) (Done action)
    in
    case action of
        Activate ->
            API.pauseExperiment handlers False

        Archive archived ->
            API.archiveExperiment handlers archived

        Cancel ->
            API.cancelExperiment handlers

        Kill ->
            API.killExperiment handlers

        Pause ->
            API.pauseExperiment handlers True


update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg model session =
    case model.model of
        Loading ->
            case msg of
                GotExperiment _ (Ok exp) ->
                    let
                        expModel =
                            initInternal exp
                    in
                    ( { model | model = WithExperiment expModel }, Cmd.none, Nothing )

                -- Error handling.
                GotCriticalError error ->
                    ( model, Cmd.none, Comm.Error error |> Just )

                GotAPIError _ err ->
                    let
                        _ =
                            -- TODO(jgevirtz): Report error to user.
                            Debug.log "Failed to load trial" err
                    in
                    ( { model | model = LoadFailed }, Cmd.none, Nothing )

                _ ->
                    ( model, Cmd.none, Nothing )

        LoadFailed ->
            case msg of
                Tick ->
                    ( model, getSummaryCmd Nothing model.id, Nothing )

                GotExperiment _ (Ok exp) ->
                    let
                        expModel =
                            initInternal exp
                    in
                    ( { model | model = WithExperiment expModel }, Cmd.none, Nothing )

                GotCriticalError error ->
                    ( model, Cmd.none, Comm.Error error |> Just )

                _ ->
                    ( model, Cmd.none, Nothing )

        WithExperiment m ->
            let
                ( newModel, cmd, outMsg ) =
                    case msg of
                        NoOp ->
                            ( m, Cmd.none, Nothing )

                        Do action ->
                            case m.pendingAction of
                                -- Only allow one action to be pending at a time.
                                Just _ ->
                                    ( m, Cmd.none, Nothing )

                                Nothing ->
                                    ( { m | pendingAction = Just action }
                                    , actionCommand action m.experiment.id
                                    , Nothing
                                    )

                        Tick ->
                            ( m, getSummaryCmd Nothing model.id, Nothing )

                        GotExperiment maybeAction (Ok exp) ->
                            let
                                pendingAction =
                                    -- We can get a timed update while waiting for an action's
                                    -- update to come back, so only clear the pending action when we
                                    -- get the actual corresponding update back.
                                    if maybeAction == m.pendingAction then
                                        Nothing

                                    else
                                        m.pendingAction
                            in
                            ( { m | experiment = exp, pendingAction = pendingAction }, Cmd.none, Nothing )

                        GotExperiment _ (Err e) ->
                            let
                                _ =
                                    Debug.log "Failed to load experiment" e
                            in
                            -- TODO(jgevirtz): Report error to user.
                            ( m, Cmd.none, Nothing )

                        PlotMsg pm ->
                            ( { m | plotModel = Plot.update pm m.plotModel }, Cmd.none, Nothing )

                        ShowCheckpoint checkpoint ->
                            let
                                ( s, c ) =
                                    MSCP.openCheckpoint m.experiment checkpoint

                                command =
                                    Cmd.map CheckpointModalMsg c
                            in
                            ( { m | checkpointModalState = s }, command, Nothing )

                        ShowConfig ->
                            let
                                ( s, c ) =
                                    MSC.openConfig m.experiment.config

                                command =
                                    Cmd.map ConfigModalMsg c
                            in
                            ( { m | configModalState = s }, command, Nothing )

                        NewTrialsTableState ts ->
                            ( { m | trialsTableState = ts }, Cmd.none, Nothing )

                        SendOut out ->
                            ( m, Cmd.none, Just out )

                        Done action _ ->
                            ( m
                            , getSummaryCmd (Just action) model.id
                            , Nothing
                            )

                        ForkExperiment ->
                            let
                                ( s, c ) =
                                    MCE.openForFork m.experiment

                                command =
                                    Cmd.map CreateExpModalMsg c
                            in
                            ( { m | createExpModalState = s }, command, Nothing )

                        CheckpointModalMsg subMsg ->
                            let
                                ( s, c ) =
                                    MSCP.update subMsg

                                command =
                                    Cmd.map CheckpointModalMsg c
                            in
                            ( { m | checkpointModalState = s }, command, Nothing )

                        ConfigModalMsg subMsg ->
                            let
                                ( s, c ) =
                                    MSC.update m.configModalState subMsg

                                command =
                                    Cmd.map ConfigModalMsg c
                            in
                            ( { m | configModalState = s }, command, Nothing )

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
                                        [ cmd1 |> Cmd.map CreateExpModalMsg
                                        , cmd2
                                        ]
                            in
                            ( newModel2, commands, outMsg2 )

                        GotTensorBoardLaunchCycleMsg tbLaunchCycle ->
                            let
                                ( c, om ) =
                                    processTensorBoardLaunchCycleMsg tbLaunchCycle
                            in
                            ( m
                            , c
                            , om
                            )

                        ButtonMsg btnID btnMsg ->
                            let
                                ( newBtnModel, btnCmd ) =
                                    getBtnModel m btnID
                                        |> Button.update btnMsg

                                updatedModel =
                                    updateBtnModel m btnID newBtnModel
                            in
                            ( updatedModel, btnCmd, Nothing )

                        ShowLess n ->
                            ( { m | numTrialsToShow = m.numTrialsToShow - n }, Cmd.none, Nothing )

                        ShowMore n ->
                            ( { m | numTrialsToShow = m.numTrialsToShow + n }, Cmd.none, Nothing )

                        ShowAll ->
                            let
                                nTrials =
                                    m.experiment.trials
                                        |> Maybe.withDefault []
                                        |> List.length
                            in
                            ( { m | numTrialsToShow = nTrials }, Cmd.none, Nothing )

                        -- Error handling.
                        GotCriticalError error ->
                            ( m, Cmd.none, Comm.Error error |> Just )

                        GotAPIError action error ->
                            let
                                _ =
                                    -- TODO(jgevirtz): Report error to user.
                                    Debug.log "Got error while performing action" ( error, action )
                            in
                            ( m, Cmd.none, Nothing )
            in
            ( { model | model = WithExperiment newModel }, cmd, outMsg )


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
    -> ExperimentModel
    -> Session
    -> ( ExperimentModel, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
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



---- VIEW


titleView : ExperimentModel -> H.Html Msg
titleView model =
    let
        parents =
            [ ( Route.toString <| Route.ExperimentList Route.defaultExperimentListOptions
              , "Experiments"
              )
            ]

        currentPageEl =
            H.span
                [ HA.title model.experiment.description ]
                [ H.text
                    ("Experiment "
                        ++ String.fromInt model.experiment.id
                        ++ ": "
                        ++ model.experiment.description
                    )
                ]
    in
    H.nav [ class "bg-blue-200 p-4" ]
        [ Page.Common.breadcrumb parents currentPageEl
        ]


plotsView : ExperimentModel -> Session -> H.Html Msg
plotsView model session =
    let
        plot =
            case model.experiment.validationHistory of
                Just history ->
                    let
                        start =
                            Time.posixToMillis model.experiment.startTime

                        processHistoryItem x =
                            case x.validationError of
                                Nothing ->
                                    Nothing

                                Just val ->
                                    Just
                                        { trial = x.trialId
                                        , endTime = x.endTime
                                        , elapsedTime = toFloat (Time.posixToMillis x.endTime - start) / 1000
                                        , metric = val
                                        }

                        plotTitle =
                            let
                                prefix =
                                    "Best validation metric"
                            in
                            case Utils.searcherValidationMetricName model.experiment.config of
                                Just validationMetric ->
                                    prefix ++ ": " ++ validationMetric

                                Nothing ->
                                    -- This shouldn't happen as validation metric is a required parameter.
                                    prefix
                    in
                    Plot.view
                        (plotConfig session plotTitle)
                        model.plotModel
                        (List.filterMap processHistoryItem history)

                Nothing ->
                    H.text ""
    in
    div [ class "flex flex-col flex-grow pt-4 lg:pt-0 lg:px-4 lg:border-l" ]
        [ div
            [ class "relative w-full h-full", HA.style "min-height" "300px" ]
            [ plot ]
        , H.ul [ HA.class "horizontal-list" ]
            [ -- Only show the scale option if there are at least three data points.
              if Maybe.Extra.unwrap 0 List.length model.experiment.validationHistory >= 3 then
                H.li []
                    [ H.text "Scale: "
                    , Plot.scaleSelectorView PlotMsg
                    ]

              else
                H.text ""
            ]
        ]


confirmableActionBtn : ExperimentModel -> ButtonID -> String -> String -> Action -> H.Html Msg
confirmableActionBtn model btnID btnText prompt action =
    let
        btnModel =
            getBtnModel model btnID

        config =
            btnModel.config

        updatedConfig =
            { config | text = btnText, isPending = model.pendingAction == Just action }
    in
    Button.view { btnModel | config = updatedConfig, promptText = prompt }
        (ButtonMsg btnID)


infoView : ExperimentModel -> Session -> H.Html Msg
infoView model session =
    let
        -- Set up action buttons.
        mkActionButton text action =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg (Do action)
                , bgColor = "blue"
                , fgColor = "white"
                , isActive = True
                , isPending = model.pendingAction == Just action
                , style = Page.Common.TextOnly
                , text = text
                }

        mkButton text msg =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg msg
                , bgColor = "blue"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = text
                }

        activateButton =
            mkActionButton "Activate" Activate

        archiveButton =
            mkActionButton
                (if model.experiment.archived then
                    "Unarchive"

                 else
                    "Archive"
                )
                (Archive (not model.experiment.archived))

        cancelButton =
            let
                prompt =
                    "Are you sure you want to cancel experiment "
                        ++ String.fromInt model.experiment.id
                        ++ "?"
            in
            confirmableActionBtn model CancelBtn "Cancel" prompt Cancel

        forkButton =
            mkButton "Fork" ForkExperiment

        openTensorBoardButtonView =
            if TensorBoard.experimentHasMetrics model.experiment then
                [ mkButton
                    "TensorBoard"
                    (TensorBoard.AccessTensorBoard
                        (Types.FromExperimentIds [ model.experiment.id ])
                        |> GotTensorBoardLaunchCycleMsg
                    )
                ]

            else
                []

        killButton =
            let
                prompt =
                    "Are you sure you want to kill experiment  "
                        ++ String.fromInt model.experiment.id
                        ++ "?"
            in
            confirmableActionBtn model KillBtn "Kill" prompt Kill

        pauseButton =
            mkActionButton "Pause" Pause

        includedButtons =
            case model.experiment.state of
                Types.Active ->
                    [ forkButton, pauseButton, cancelButton, killButton ]
                        ++ openTensorBoardButtonView

                Types.Canceled ->
                    [ forkButton, archiveButton ]

                Types.Completed ->
                    [ forkButton, archiveButton ]
                        ++ openTensorBoardButtonView

                Types.Error ->
                    [ forkButton, archiveButton ]

                Types.Paused ->
                    [ forkButton, activateButton, cancelButton, killButton ]
                        ++ openTensorBoardButtonView

                Types.StoppingCanceled ->
                    [ forkButton, killButton ]

                Types.StoppingCompleted ->
                    [ forkButton, killButton ]
                        ++ openTensorBoardButtonView

                Types.StoppingError ->
                    [ forkButton, killButton ]

        -- Set up informational table.
        mkLabel text =
            H.td [ class "font-bold p-2" ] [ H.text text ]

        mkValue =
            H.td [ class "p-2" ]

        stateElem =
            H.span []
                [ Page.Common.runStateToSpan model.experiment.state
                , H.text <|
                    if model.experiment.archived then
                        " (archived)"

                    else
                        ""
                ]

        progressElem =
            model.experiment.progress
                |> Maybe.Extra.unwrap "N/A" ((*) 100 >> Formatting.toPct)
                |> H.text

        startTimeElem =
            model.experiment.startTime
                |> Formatting.posixToString session.zone
                |> H.text

        endTimeElem =
            model.experiment.endTime
                |> Maybe.Extra.unwrap "N/A" (Formatting.posixToString session.zone)
                |> H.text

        maxSlotsElem =
            model.experiment.maxSlots
                |> Maybe.Extra.unwrap "Unlimited" String.fromInt
                |> H.text

        configElem =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg ShowConfig
                , bgColor = "green"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = "Show"
                }

        modelDefElem =
            let
                url =
                    API.buildUrl [ "experiments", String.fromInt model.experiment.id, "model_def" ] []
            in
            Page.Common.buttonCreator
                { action = Page.Common.OpenUrl True url
                , bgColor = "yellow"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = "Download"
                }

        ( bestValidationElem, bestCheckpointElem ) =
            case getBestCheckpoint model.experiment of
                Just checkpoint ->
                    let
                        batchesPerStep =
                            Utils.batchesPerStep model.experiment.config

                        buttonLabel =
                            "Trial "
                                ++ String.fromInt checkpoint.trialId
                                ++ " Batch "
                                ++ String.fromInt (checkpoint.stepId * batchesPerStep)

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
                            Utils.searcherValidationMetricName model.experiment.config
                                |> Maybe.Extra.unwrap "" (\name -> " (" ++ name ++ ")")

                        metricValue =
                            Maybe.withDefault 0 checkpoint.validationMetric
                                |> Formatting.validationFormat
                    in
                    ( [ mkLabel ("Best validation" ++ metricType)
                      , mkValue [ H.text metricValue ]
                      ]
                    , [ mkLabel "Best checkpoint", mkValue [ button ] ]
                    )

                Nothing ->
                    ( [], [] )

        tableRows =
            [ [ mkLabel "State", mkValue [ stateElem ] ]
            , [ mkLabel "Progress", mkValue [ progressElem ] ]
            , [ mkLabel "Start time", mkValue [ startTimeElem ] ]
            , [ mkLabel "End time", mkValue [ endTimeElem ] ]
            , [ mkLabel "Max slots", mkValue [ maxSlotsElem ] ]
            , bestValidationElem
            , bestCheckpointElem
            , [ mkLabel "Configuration", mkValue [ configElem ] ]
            , [ mkLabel "Model definition", mkValue [ modelDefElem ] ]
            ]

        table =
            H.table [ class "text-sm text-gray-700 my-2" ] <| List.map (H.tr []) tableRows
    in
    div [ class "pb-2 lg:pb-0 border-b lg:border-none lg:pr-4", HA.style "min-width" "15rem" ]
        [ List.map (\item -> H.li [] [ item ]) includedButtons
            |> H.ul [ HA.class "horizontal-list" ]
        , table
        ]


topBoxView : ExperimentModel -> Session -> H.Html Msg
topBoxView model session =
    div [ class "flex w-full flex-col lg:flex-row text-sm pb-4 text-gray-700" ]
        [ infoView model session, plotsView model session ]


trialsTableViewHelper : Types.Experiment -> Table.State -> Table.Config Types.TrialSummary Msg -> Int -> H.Html Msg
trialsTableViewHelper experiment tableState tableConfig numTrialsToShow =
    let
        table =
            case experiment.trials of
                Just ((_ :: _) as trials) ->
                    trials
                        -- As secondary and tertiary sorting order we use number of steps
                        -- (descending) and time created (descending), respectively.
                        |> List.sortBy (\t -> ( -t.numSteps, -(Time.posixToMillis t.startTime) ))
                        |> Table.view tableConfig tableState (Just numTrialsToShow)

                _ ->
                    H.text ""
    in
    Page.Common.section "Trials" [ table ]


trialsTableView : ExperimentModel -> Session -> H.Html Msg
trialsTableView model session =
    Html.Lazy.lazy4 trialsTableViewHelper
        model.experiment
        model.trialsTableState
        (trialsTableConfig session model)
        model.numTrialsToShow


view : Model -> Session -> Html Msg
view model session =
    let
        body =
            case model.model of
                Loading ->
                    [ Page.Common.centeredLoadingWidget ]

                LoadFailed ->
                    [ Page.Common.bigMessage ("Failed to load experiment " ++ String.fromInt model.id ++ ".") ]

                WithExperiment m ->
                    [ titleView m
                    , div
                        [ class "w-full text-sm p-4" ]
                        [ topBoxView m session
                        , trialsTableView m session
                        , MSCP.view m.checkpointModalState session
                            |> H.map CheckpointModalMsg
                        , MSC.view m.configModalState
                            |> H.map ConfigModalMsg
                        , MCE.view m.createExpModalState
                            |> H.map CreateExpModalMsg
                        ]
                    ]
    in
    div [ class "w-full" ] body
