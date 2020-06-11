module Page.ExperimentList exposing
    ( Model
    , Msg(..)
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
import Components.DropdownSelect as DS exposing (DropdownConfig, DropdownState, defaultInitialState, dropDownSelect)
import Components.LabelWidget as LW
import Components.Table as Table exposing (Status(..))
import Components.Table.Custom exposing (datetimeCol, limitedWidthStringColWithIcon, percentCol, runStateCol, tableCustomizations)
import Dict
import EverySet
import Formatting exposing (runStateToString)
import Html as H
import Html.Attributes as HA
import Html.Events as HE
import Json.Decode as D
import Json.Encode as E
import List.Extra
import Maybe.Extra
import Modals.CreateExperiment as CE
import Modules.TensorBoard as TensorBoard
import Page.Common exposing (centeredLoadingWidget)
import Result.Extra exposing (unpack, unwrap)
import Route
import Session exposing (Session)
import Set
import Time
import Types
import Utils


type ExperimentUpdate
    = Pause
    | Archive
    | ChangeLabel LW.LabelEdit
    | Kill


type alias BatchOperation =
    { isPossible : Types.ExperimentResult -> Bool
    , isComplete : Types.ExperimentResult -> Bool
    , toCmd : Types.ID -> Cmd Msg
    , operation : ExperimentUpdate
    }


batchOperationPending : BatchOperation -> LoadedModel -> Bool
batchOperationPending batchOperation model =
    case model.pendingBatchRequest of
        Just ( pbo, ids ) ->
            if pbo.operation /= batchOperation.operation then
                False

            else
                batchOperationComplete pbo model ids
                    |> not

        Nothing ->
            False


batchOperationPossible : BatchOperation -> LoadedModel -> Bool
batchOperationPossible batchOperation model =
    let
        selected =
            model.experiments
                |> Maybe.withDefault []
                |> List.filterMap
                    (\tr ->
                        let
                            isSelected =
                                tr.selected

                            isVisible =
                                filterExperimentResults model tr
                        in
                        if isSelected && isVisible then
                            Just tr.experimentResult

                        else
                            Nothing
                    )
    in
    Maybe.Extra.isNothing model.pendingBatchRequest
        -- Not strictly necessary, but helps ensure consistency in the UI by disabling batch
        -- requests when a TB is being opened just as batch operations are disabled when
        -- another batch operation is in progress.
        |> (&&) (Maybe.Extra.isNothing model.pendingTensorBoard)
        |> (&&) (not (List.isEmpty selected))
        |> (&&) (List.all batchOperation.isPossible selected)


batchOperationComplete : BatchOperation -> LoadedModel -> Set.Set Types.ID -> Bool
batchOperationComplete batchOperation model workload =
    let
        experimentResults =
            model.experiments
                |> Maybe.withDefault []
                |> List.filterMap
                    (\tr ->
                        let
                            id =
                                unpack .id .id tr.experimentResult
                        in
                        if Set.member id workload then
                            Just tr.experimentResult

                        else
                            Nothing
                    )
    in
    List.all batchOperation.isComplete experimentResults


performBatchOperation : BatchOperation -> List Types.ExperimentResult -> Maybe (Cmd Msg)
performBatchOperation batchOperation experimentResults =
    if not (List.all batchOperation.isPossible experimentResults) then
        Nothing

    else
        let
            ids =
                List.map (unpack .id .id) experimentResults
        in
        List.map batchOperation.toCmd ids
            |> Cmd.batch
            |> Just


batchKill : BatchOperation
batchKill =
    let
        toCmd id =
            API.killExperiment
                (requestHandlersAction Kill id (always << always NoOp))
                id
    in
    { isPossible =
        Result.Extra.unwrap
            False
            (\e -> e.state == Types.Active || e.state == Types.Paused)
    , isComplete =
        Result.Extra.unwrap
            False
            (\e -> e.state /= Types.Active)
    , toCmd = toCmd
    , operation = Kill
    }


batchPause : BatchOperation
batchPause =
    let
        toCmd id =
            API.pauseExperiment
                (requestHandlersAction Pause id (always << always NoOp))
                True
                id
    in
    { isPossible =
        Result.Extra.unwrap
            False
            (\e -> e.state == Types.Active)
    , isComplete =
        Result.Extra.unwrap
            False
            (\e -> e.state == Types.Paused)
    , toCmd = toCmd
    , operation = Pause
    }


batchArchive : BatchOperation
batchArchive =
    let
        toCmd id =
            API.archiveExperiment
                (requestHandlersAction Archive id (always << always NoOp))
                True
                id
    in
    { isPossible =
        Result.Extra.unwrap
            False
            (\e ->
                (case e.state of
                    Types.Canceled ->
                        True

                    Types.Completed ->
                        True

                    Types.Error ->
                        True

                    _ ->
                        False
                )
                    |> (&&) (not e.archived)
            )
    , isComplete =
        Result.Extra.unwrap
            False
            (\e -> e.archived)
    , toCmd = toCmd
    , operation = Archive
    }


entriesShowIncrement : Int
entriesShowIncrement =
    50


type alias TableExperiment =
    { experimentResult : Types.ExperimentResult
    , labelWidgetState : LW.State
    , selected : Bool
    }


type alias FilterState =
    { descriptionFilter : String
    , showArchived : Bool
    , labelDropdownState : DropdownState String
    , stateDropdownState : DropdownState Types.RunState
    , ownerDropdownState : DropdownState Types.User
    }


type ButtonID
    = ArchiveBtn
    | PauseBtn
    | KillBtn


getBtnModel : LoadedModel -> ButtonID -> Button.Model Msg
getBtnModel model id =
    case id of
        ArchiveBtn ->
            model.archiveBtn

        PauseBtn ->
            model.pauseBtn

        KillBtn ->
            model.killBtn


updateBtnModel : LoadedModel -> ButtonID -> Button.Model Msg -> LoadedModel
updateBtnModel model id btnModel =
    case id of
        ArchiveBtn ->
            { model | archiveBtn = btnModel }

        PauseBtn ->
            { model | pauseBtn = btnModel }

        KillBtn ->
            { model | killBtn = btnModel }


type alias LoadedModel =
    { tableSortState : Table.State
    , experiments : Maybe (List TableExperiment)
    , createExpModalState : CE.Model
    , filterState : FilterState
    , archiveBtn : Button.Model Msg
    , pauseBtn : Button.Model Msg
    , killBtn : Button.Model Msg
    , pendingBatchRequest : Maybe ( BatchOperation, Set.Set Types.ID )
    , pendingTensorBoard : Maybe (Set.Set Types.ID)
    , numEntriesToShow : Int
    }


type Model
    = Loading (Maybe FilterState) (Maybe Table.State) Route.ExperimentListOptions
    | Loaded LoadedModel


requestHandlersGet : (body -> Msg) -> API.RequestHandlers Msg body
requestHandlersGet onSuccess =
    { onSuccess = onSuccess
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIErrorGetExps
    }


requestHandlersAction : ExperimentUpdate -> Types.ID -> (Types.ID -> body -> Msg) -> API.RequestHandlers Msg body
requestHandlersAction updateType id onSuccess =
    { onSuccess = onSuccess id
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIErrorAction updateType id
    }


labelWidgetConfig : LW.Config Msg
labelWidgetConfig =
    { toMsg = UpdateLabelWidgetState
    , toLabelEditMsg = LabelEditRequest
    , displayAtMost = Just 3
    , toLabelClickedMsg = LabelClicked
    , maxLabelLength = 15
    }


statesList : List Types.RunState
statesList =
    [ Types.Active
    , Types.Canceled
    , Types.Completed
    , Types.Error
    , Types.Paused
    , Types.StoppingCanceled
    , Types.StoppingCompleted
    , Types.StoppingError
    ]


{-| defaultFilterState generates the FilterState with nothing selected except for the
currently-authenticated user.
-}
defaultFilterState : Session -> List Types.ExperimentResult -> FilterState
defaultFilterState session experiments =
    let
        owners =
            extractOwnerList experiments

        labels =
            extractLabels experiments

        selectedOwners =
            Maybe.Extra.unwrap
                EverySet.empty
                (EverySet.singleton << .user)
                session.user
    in
    { descriptionFilter = ""
    , showArchived = False
    , labelDropdownState = defaultInitialState labels
    , stateDropdownState = defaultInitialState statesList
    , ownerDropdownState =
        defaultInitialState owners
            |> DS.selectItems selectedOwners
    }


initTableState : Maybe Table.State -> Route.ExperimentListOptions -> Table.State
initTableState tableState options =
    let
        sortReversed =
            Maybe.withDefault
                False
                options.sortReversed

        default =
            Maybe.withDefault
                (Table.initialSortWithReverse "state" sortReversed)
                tableState
    in
    Maybe.Extra.unwrap
        default
        (\col -> Table.initialSortWithReverse col sortReversed)
        options.sort


initFilterState : Session -> Maybe FilterState -> Route.ExperimentListOptions -> List Types.ExperimentResult -> FilterState
initFilterState session filterState options experiments =
    let
        owners =
            extractOwnerList experiments

        default =
            Maybe.withDefault
                (defaultFilterState session experiments)
                filterState

        updateState : List a -> DS.DropdownState a -> DS.DropdownState a
        updateState selected =
            DS.clear >> DS.selectItems (EverySet.fromList selected)
    in
    { default
        | descriptionFilter =
            Maybe.withDefault default.descriptionFilter options.description
        , showArchived =
            Maybe.withDefault default.showArchived options.archived
        , labelDropdownState =
            default.labelDropdownState
                |> Maybe.Extra.unwrap
                    identity
                    updateState
                    options.labels
        , stateDropdownState =
            default.stateDropdownState
                |> Maybe.Extra.unwrap
                    identity
                    updateState
                    options.states
        , ownerDropdownState =
            case Maybe.map EverySet.fromList options.users of
                Just usernames ->
                    updateState
                        (List.filter (\u -> EverySet.member u.username usernames) owners)
                        default.ownerDropdownState

                Nothing ->
                    default.ownerDropdownState
    }


expResultToTableExp : Types.ExperimentResult -> TableExperiment
expResultToTableExp er =
    { experimentResult = er
    , labelWidgetState = unpack .id .id er |> LW.init
    , selected = False
    }


{-| Apply sorting before the rendering the table so that after the primary sorting
is done by the table, elements are sorted secondarily by start time.
-}
applyInitialSort : List TableExperiment -> List TableExperiment
applyInitialSort experiments =
    List.sortBy (Result.Extra.unwrap 0 ((*) -1 << Time.posixToMillis << .startTime) << .experimentResult) experiments


initLoaded : Session -> FilterState -> Table.State -> List Types.ExperimentResult -> ( LoadedModel, Cmd Msg )
initLoaded session filterState tableState experiments =
    let
        baseBatchButton text confirmBtnText bgColor msg =
            { needsConfirmation = True
            , confirmBtnText = confirmBtnText
            , dismissBtnText = "Dismiss"
            , promptOpen = False
            , promptTitle = "Confirm Action"
            , promptText = "Are you sure?"
            , config =
                { action = Page.Common.SendMsg msg
                , bgColor = bgColor
                , fgColor = "white"
                , isActive = False
                , isPending = False
                , style = Page.Common.TextOnly
                , text = text
                }
            }

        model =
            { tableSortState = tableState
            , experiments =
                List.map expResultToTableExp experiments
                    |> applyInitialSort
                    |> Just
            , createExpModalState = CE.init
            , archiveBtn = baseBatchButton "Archive Selected" "Archive" "orange" (DoBatchOperation batchArchive)
            , pauseBtn = baseBatchButton "Pause Selected" "Pause" "orange" (DoBatchOperation batchPause)
            , killBtn = baseBatchButton "Kill Selected" "Kill" "red" (DoBatchOperation batchKill)
            , pendingBatchRequest = Nothing
            , pendingTensorBoard = Nothing
            , filterState = filterState
            , numEntriesToShow = entriesShowIncrement
            }

        cmd =
            updateQueryParameters session model
    in
    ( model
    , cmd
    )


init : Maybe Model -> Route.ExperimentListOptions -> ( Model, Cmd Msg )
init previousModel options =
    let
        ( filterState, tableState ) =
            case previousModel of
                Just (Loaded m) ->
                    ( Just m.filterState, Just m.tableSortState )

                _ ->
                    ( Nothing, Nothing )
    in
    ( Loading filterState tableState options
    , pollExperiments
    )


subscriptions : Model -> Sub Msg
subscriptions model =
    case model of
        Loading _ _ _ ->
            Sub.none

        Loaded m ->
            let
                tickSub =
                    Time.every 5000 (\_ -> Tick)

                modalSub =
                    CE.subscriptions m.createExpModalState
                        |> Sub.map CreateExpModalMsg
            in
            Sub.batch [ tickSub, modalSub ]


type FilterUpdate
    = LabelFilter (DropdownState String)
    | RunStateFilter (DropdownState Types.RunState)
    | OwnerFilter (DropdownState Types.User)
    | DescriptionFilter String
    | ShowArchivedFilter Bool
    | ResetFilters


type Msg
    = NoOp
    | GotExperiments (List Types.ExperimentResult)
    | NewTableState Table.State
    | GotFilterUpdate FilterUpdate
    | SendOut (Comm.OutMessage OutMsg)
    | TableCheckboxClicked Int Bool
    | ToggleSelectAll Bool
    | Tick
    | UpdateLabelWidgetState LW.State (Maybe (Cmd Msg))
    | LabelEditRequest LW.State LW.LabelEdit
    | LabelChanged Types.ID ()
    | LabelClicked String
    | OwnerClicked Types.User
      -- Batch operations.
    | DoBatchOperation BatchOperation
      -- Forking.
    | ForkExperiment Types.Experiment
    | CreateExpModalMsg CE.Msg
      -- TensorBoards. Opening/launching a TensorBoard is a multi-step process
      -- that is routed through GotTensorBoardLaunchCycleMsg.
    | GotTensorBoardLaunchCycleMsg (Set.Set Types.ID) TensorBoard.TensorBoardLaunchCycleMsg
    | OpenTBOverSelectedExperiments
      -- Table control.
    | ShowLess Int
    | ShowMore Int
    | ShowAll
      -- Errors.
    | GotCriticalError Comm.SystemError
    | GotAPIErrorAction ExperimentUpdate Types.ID API.APIError
    | GotAPIErrorGetExps API.APIError
      -- Buttons
    | ButtonMsg ButtonID Button.Msg


type OutMsg
    = SetCriticalError String


pollExperiments : Cmd Msg
pollExperiments =
    requestHandlersGet GotExperiments
        |> API.pollExperiments


{-| Update the filter state using the given function. The function that is expected as the first
parameter returns a tuple. The second element of the tuple is expected to be the updated filter
state. The first element is a boolean value indicating whether the filter update represents a
change that will alter the experiments that are shown to the user. It is possible to get a value
of False if, for instance, the updated FilterState corresponds only to the opening of a filter
dropdown (which changes DropSelect.DropdownState).
-}
updateFilterState : (FilterState -> ( Bool, FilterState )) -> LoadedModel -> LoadedModel
updateFilterState updateFn model =
    let
        ( selectionsChanged, updatedFilterState ) =
            updateFn model.filterState

        newModel =
            if selectionsChanged then
                toggleSelectAll False model

            else
                model
    in
    { newModel | filterState = updatedFilterState }


updateLoaded : Msg -> LoadedModel -> Session -> ( LoadedModel, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
updateLoaded msg model session =
    case msg of
        NoOp ->
            ( model, Cmd.none, Nothing )

        Tick ->
            ( model, pollExperiments, Nothing )

        ButtonMsg btnID btnMsg ->
            let
                ( newBtnModel, cmd ) =
                    getBtnModel model btnID
                        |> Button.update btnMsg

                newModel =
                    updateBtnModel model btnID newBtnModel
            in
            ( newModel, cmd, Nothing )

        GotExperiments experiments ->
            let
                owners =
                    extractOwnerList experiments

                labels =
                    extractLabels experiments

                labelDropdownState =
                    DS.setOptions labels model.filterState.labelDropdownState

                -- By default, only display experiments for session's user.
                ownerDropdownState =
                    case ( session.user, model.experiments ) of
                        -- Only do this if we are loading experiments for the first time. Otherwise,
                        -- we would be overriding changes to the owner filter made by the user.
                        ( Just { user }, Nothing ) ->
                            DS.setOptions owners model.filterState.ownerDropdownState
                                |> DS.selectItems (EverySet.fromList [ { username = user.username, id = user.id } ])

                        _ ->
                            DS.setOptions owners model.filterState.ownerDropdownState

                updatedExperiments =
                    updateExperiments model experiments
                        |> applyInitialSort

                newPendingBatchOperation =
                    Maybe.andThen
                        (\( pbo, ids ) ->
                            if batchOperationComplete pbo model ids then
                                Nothing

                            else
                                Just ( pbo, ids )
                        )
                        model.pendingBatchRequest

                -- Did any filter selections change as a result of handling this GotExperiments message?
                selectionsChanged =
                    List.any
                        identity
                        [ DS.selectionsDiffer
                            labelDropdownState
                            model.filterState.labelDropdownState
                        , DS.selectionsDiffer
                            ownerDropdownState
                            model.filterState.ownerDropdownState
                        ]

                filterStateUpdateFn fs =
                    ( selectionsChanged
                    , { fs
                        | labelDropdownState = labelDropdownState
                        , ownerDropdownState = ownerDropdownState
                      }
                    )

                newModel =
                    { model
                        | experiments = Just updatedExperiments
                        , pendingBatchRequest = newPendingBatchOperation
                    }
                        |> updateFilterState filterStateUpdateFn
            in
            ( newModel
            , Cmd.none
            , Nothing
            )

        SendOut out ->
            ( model, Cmd.none, Just out )

        NewTableState state ->
            let
                newModel =
                    { model | tableSortState = state }
            in
            ( newModel, updateQueryParameters session newModel, Nothing )

        TableCheckboxClicked id _ ->
            let
                mapper : TableExperiment -> TableExperiment
                mapper tableExp =
                    if unpack .id .id tableExp.experimentResult == id then
                        { tableExp | selected = not tableExp.selected }

                    else
                        tableExp

                updated =
                    Maybe.map (List.map mapper) model.experiments

                newModel =
                    { model | experiments = updated }
            in
            ( newModel, Cmd.none, Nothing )

        ToggleSelectAll val ->
            ( toggleSelectAll val model, Cmd.none, Nothing )

        GotFilterUpdate filterUpdate ->
            let
                filterStateUpdateFn fs =
                    case filterUpdate of
                        LabelFilter s ->
                            ( DS.selectionsDiffer s fs.labelDropdownState
                            , { fs | labelDropdownState = s }
                            )

                        RunStateFilter s ->
                            ( DS.selectionsDiffer s fs.stateDropdownState
                            , { fs | stateDropdownState = s }
                            )

                        OwnerFilter s ->
                            ( DS.selectionsDiffer s fs.ownerDropdownState
                            , { fs | ownerDropdownState = s }
                            )

                        DescriptionFilter d ->
                            ( True, { fs | descriptionFilter = d } )

                        ShowArchivedFilter s ->
                            ( True, { fs | showArchived = s } )

                        ResetFilters ->
                            ( True
                            , Maybe.Extra.unwrap [] (List.map .experimentResult) model.experiments
                                |> defaultFilterState session
                            )

                newModel =
                    updateFilterState filterStateUpdateFn model
            in
            ( newModel
            , updateQueryParameters session newModel
            , Nothing
            )

        UpdateLabelWidgetState state maybeCmd ->
            let
                newModel =
                    updateLabelWidgetState state model
            in
            ( newModel, Maybe.withDefault Cmd.none maybeCmd, Nothing )

        LabelEditRequest state edit ->
            let
                newModel =
                    updateLabelWidgetState state model
            in
            ( newModel, addOrRemoveLabel edit, Nothing )

        LabelChanged _ _ ->
            ( model, pollExperiments, Nothing )

        LabelClicked label ->
            let
                filterStateUpdateFn fs =
                    let
                        labelDropdownState =
                            DS.selectItems (EverySet.fromList [ label ]) fs.labelDropdownState

                        selectionsChanged =
                            DS.selectionsDiffer
                                labelDropdownState
                                fs.labelDropdownState
                    in
                    ( selectionsChanged
                    , { fs
                        | labelDropdownState = labelDropdownState
                      }
                    )

                newModel =
                    updateFilterState filterStateUpdateFn model
            in
            ( newModel
            , updateQueryParameters session newModel
            , Nothing
            )

        OwnerClicked owner ->
            let
                filterStateUpdateFn fs =
                    let
                        ownerDropdownState =
                            DS.selectItems (EverySet.fromList [ owner ]) fs.ownerDropdownState

                        selectionsChanged =
                            DS.selectionsDiffer
                                ownerDropdownState
                                fs.ownerDropdownState
                    in
                    ( selectionsChanged
                    , { fs | ownerDropdownState = ownerDropdownState }
                    )

                newModel =
                    updateFilterState filterStateUpdateFn model
            in
            ( newModel
            , updateQueryParameters session newModel
            , Nothing
            )

        DoBatchOperation batchOperation ->
            let
                experiments =
                    model.experiments
                        |> Maybe.withDefault []
                        -- Make sure experiment is visible selected.
                        |> List.filter
                            (\e ->
                                e.selected && filterExperimentResults model e
                            )
                        |> List.map .experimentResult

                cmd =
                    performBatchOperation batchOperation experiments

                experimentIds =
                    List.map (unpack .id .id) experiments

                newModel =
                    { model
                        | pendingBatchRequest = Just ( batchOperation, Set.fromList experimentIds )
                    }
            in
            ( newModel, Maybe.withDefault Cmd.none cmd, Nothing )

        ForkExperiment experiment ->
            let
                ( createExpModalState, cmd ) =
                    CE.openForFork experiment
            in
            ( { model | createExpModalState = createExpModalState }
            , cmd |> Cmd.map CreateExpModalMsg
            , Nothing
            )

        CreateExpModalMsg subMsg ->
            let
                ( newCreateExpModelState, cmd1, outMsg1 ) =
                    CE.update subMsg model.createExpModalState

                ( newModel1, cmd2, outMsg2 ) =
                    handleCreateExperimentModalMsg outMsg1 model session

                newModel2 =
                    { newModel1 | createExpModalState = newCreateExpModelState }

                commands =
                    Cmd.batch
                        [ cmd1 |> Cmd.map CreateExpModalMsg
                        , cmd2
                        ]
            in
            ( newModel2, commands, outMsg2 )

        GotTensorBoardLaunchCycleMsg ids tbLaunchCycle ->
            let
                ( status, cmd ) =
                    TensorBoard.handleTensorBoardLaunchCycleMsg
                        tbLaunchCycle
                        (GotTensorBoardLaunchCycleMsg ids)

                ( newModel, outMsg ) =
                    case status of
                        Types.RequestSettled _ ->
                            ( { model | pendingTensorBoard = Nothing }, Nothing )

                        Types.RequestPending ->
                            ( model, Nothing )

                        Types.RequestFailed (TensorBoard.API _) ->
                            -- TODO(jgevirtz): Report error to user.
                            ( model, Nothing )

                        Types.RequestFailed (TensorBoard.Critical e) ->
                            ( model, Comm.Error e |> Just )
            in
            ( newModel
            , cmd
            , outMsg
            )

        OpenTBOverSelectedExperiments ->
            let
                expIds =
                    filterSelectedAvailableExperiments model
                        |> List.map .id
            in
            if List.length expIds > 0 then
                let
                    ( _, cmd ) =
                        TensorBoard.handleTensorBoardLaunchCycleMsg
                            (Types.FromExperimentIds expIds
                                |> TensorBoard.AccessTensorBoard
                            )
                            (Set.fromList expIds |> GotTensorBoardLaunchCycleMsg)

                    newModel =
                        { model
                            | pendingTensorBoard = Just (Set.fromList expIds)
                        }
                in
                ( newModel, cmd, Nothing )

            else
                ( model, Cmd.none, Nothing )

        ShowLess n ->
            ( { model | numEntriesToShow = model.numEntriesToShow - n }, Cmd.none, Nothing )

        ShowMore n ->
            ( { model | numEntriesToShow = model.numEntriesToShow + n }, Cmd.none, Nothing )

        ShowAll ->
            let
                nEntries =
                    model.experiments
                        |> Maybe.withDefault []
                        |> List.length
            in
            ( { model | numEntriesToShow = nEntries }, Cmd.none, Nothing )

        GotCriticalError crit ->
            ( model, Cmd.none, Comm.Error crit |> Just )

        GotAPIErrorAction u _ _ ->
            let
                -- TODO(jgevirtz): Report error to user.
                _ =
                    Debug.log "Failed to perform update" u
            in
            ( model, Cmd.none, Nothing )

        GotAPIErrorGetExps error ->
            let
                -- TODO(jgevirtz): Report error to user.
                _ =
                    Debug.log "Failed to get experiments" error
            in
            ( model, Cmd.none, Nothing )


update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg model session =
    let
        errorMessage =
            "An unknown error occurred while retrieving experiments.  Try refreshing the page.  If the error persists, contact us."
    in
    case model of
        Loading filterState tableState options ->
            case msg of
                GotExperiments experiments ->
                    let
                        ( m, cmd ) =
                            initLoaded
                                session
                                (initFilterState
                                    session
                                    filterState
                                    options
                                    experiments
                                )
                                (initTableState
                                    tableState
                                    options
                                )
                                experiments
                    in
                    ( Loaded m, cmd, Nothing )

                GotAPIErrorGetExps error ->
                    let
                        -- TODO(jgevirtz): Report error to user.
                        _ =
                            Debug.log "Failed to get experiments" error
                    in
                    ( model, Cmd.none, Just (Comm.OutMessage (SetCriticalError errorMessage)) )

                _ ->
                    ( model, Cmd.none, Nothing )

        Loaded m ->
            let
                ( lm, cmd, outMsg ) =
                    updateLoaded msg m session
            in
            ( Loaded lm, cmd, outMsg )


{-| Find and return all availabe, selected and filtered experiements.
-}
filterSelectedAvailableExperiments : LoadedModel -> List Types.Experiment
filterSelectedAvailableExperiments model =
    model.experiments
        |> Maybe.Extra.unwrap [] (List.filter (filterExperimentResults model))
        |> List.filter .selected
        |> List.map .experimentResult
        |> List.filterMap Result.toMaybe


updateQueryParameters : Session -> LoadedModel -> Cmd Msg
updateQueryParameters session m =
    let
        ( sort, sortReversed ) =
            Table.getSortState m.tableSortState

        options =
            { users =
                m.filterState.ownerDropdownState.selectedFilters
                    |> EverySet.toList
                    |> List.map .username
                    |> Just
            , labels =
                m.filterState.labelDropdownState.selectedFilters
                    |> EverySet.toList
                    |> Utils.listToMaybe
            , states =
                m.filterState.stateDropdownState.selectedFilters
                    |> EverySet.toList
                    |> Utils.listToMaybe
            , description =
                case m.filterState.descriptionFilter of
                    "" ->
                        Nothing

                    description ->
                        Just description
            , archived =
                if m.filterState.showArchived then
                    Just True

                else
                    Nothing
            , sort =
                if sort == "state" then
                    Nothing

                else
                    Just sort
            , sortReversed =
                if sortReversed then
                    Just True

                else
                    Nothing
            }
    in
    Route.toString (Route.ExperimentList options)
        |> Navigation.replaceUrl session.key


handleCreateExperimentModalMsg :
    Maybe (Comm.OutMessage CE.OutMsg)
    -> LoadedModel
    -> Session
    -> ( LoadedModel, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
handleCreateExperimentModalMsg msg model session =
    case msg of
        Just (Comm.OutMessage (CE.CreatedExperiment id)) ->
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


addOrRemoveLabel : LW.LabelEdit -> Cmd Msg
addOrRemoveLabel edit =
    let
        ( expId, label, targetValue ) =
            case edit of
                LW.Add id l ->
                    ( id, l, E.bool True )

                LW.Remove id l ->
                    ( id, l, E.null )

        innerValue =
            E.object [ ( label, targetValue ) ]

        value =
            E.object [ ( "labels", innerValue ) ]

        handlers =
            requestHandlersAction (ChangeLabel edit) expId LabelChanged
    in
    API.patchExperiment handlers value expId


updateExperiments : LoadedModel -> List Types.ExperimentResult -> List TableExperiment
updateExperiments model results =
    let
        expDict =
            case model.experiments of
                Just current ->
                    List.map (\e -> ( unpack .id .id e.experimentResult, e )) current
                        |> Dict.fromList

                Nothing ->
                    Dict.empty

        mapper : Types.ExperimentResult -> TableExperiment
        mapper r =
            let
                id =
                    unpack .id .id r
            in
            case Dict.get id expDict of
                Just state ->
                    { experimentResult = r
                    , labelWidgetState = state.labelWidgetState
                    , selected = state.selected
                    }

                Nothing ->
                    expResultToTableExp r
    in
    List.map mapper results


viewEmpty : Bool -> H.Html Msg
viewEmpty becauseOfFilters =
    let
        message =
            if becauseOfFilters then
                "No experiments matching the selected filters were found."

            else
                "No experiments have been created yet."
    in
    H.div [ HA.class "flex flex-row justify-center text-gray-600 text-2xl" ]
        [ H.text message ]


view : Model -> Session -> H.Html Msg
view model sess =
    case model of
        Loading _ _ _ ->
            centeredLoadingWidget

        Loaded m ->
            let
                empty =
                    m.experiments
                        |> Maybe.Extra.unwrap [] (List.filter (filterExperimentResults m))
                        |> List.isEmpty

                becauseOfFilters =
                    m.experiments
                        |> Maybe.Extra.unwrap False (not << List.isEmpty)
            in
            H.div [ HA.class "w-full text-sm p-4", HA.id "experimentsList" ]
                [ viewTableHead empty m
                , if empty then
                    viewEmpty becauseOfFilters

                  else
                    H.div [ HA.class "w-full overflow-x-auto" ]
                        [ viewTableBody m sess ]
                , CE.view m.createExpModalState
                    |> H.map CreateExpModalMsg
                ]


allVisibleExperimentsSelected : LoadedModel -> Bool
allVisibleExperimentsSelected model =
    case model.experiments of
        Nothing ->
            False

        Just [] ->
            False

        Just l ->
            List.filter (filterExperimentResults model) l
                |> List.all .selected


viewTableHead : Bool -> LoadedModel -> H.Html Msg
viewTableHead hideBatchOperations model =
    let
        invisible =
            if hideBatchOperations then
                HA.class "invisible"

            else
                HA.class ""

        enablePause =
            batchOperationPossible batchPause model

        enableArchive =
            batchOperationPossible batchArchive model

        anySelectedAndVisible =
            Maybe.withDefault [] model.experiments
                |> List.map (\te -> te.selected && filterExperimentResults model te)
                |> List.any identity

        enableOpenTB =
            Maybe.Extra.isNothing model.pendingTensorBoard
                -- Not strictly necessary, but helps ensure consistency in the UI by disabling TB
                -- button when a batch operation is in progress just as batch operations are
                -- disabled when another batch operation is in progress.
                |> (&&) (Maybe.Extra.isNothing model.pendingBatchRequest)
                |> (&&) anySelectedAndVisible

        enableKill =
            batchOperationPossible batchKill model

        pausing =
            batchOperationPending batchPause model

        archiving =
            batchOperationPending batchArchive model

        openingTensorBoard =
            Maybe.Extra.isJust model.pendingTensorBoard

        killing =
            batchOperationPending batchKill model

        nSelected =
            Maybe.withDefault [] model.experiments
                |> List.Extra.count (\e -> e.selected && filterExperimentResults model e)

        nSelectedHtml =
            if nSelected > 0 then
                H.span [ HA.class "text-xs" ]
                    [ "("
                        ++ String.fromInt nSelected
                        ++ " selected)"
                        |> H.text
                    ]

            else
                H.text ""
    in
    H.div [ HA.class "w-full text-sm text-gray-700 pb-5" ]
        [ Page.Common.pageHeader "Experiments"
        , H.div [ HA.class "w-full flex flex-wrap items-baseline pb-8 filters" ]
            [ H.div [ HA.class "pr-4 mb-4" ]
                [ H.input
                    [ HE.onInput (GotFilterUpdate << DescriptionFilter)
                    , HA.attribute "aria-label" "Search description"
                    , HA.class "appearance-none inline bg-transparent border-b-2 border-gray-500 py-1 focus:outline-none"
                    , HA.placeholder "Search description"
                    , HA.type_ "text"
                    , HA.value model.filterState.descriptionFilter
                    ]
                    []
                ]
            , H.div [ HA.class "px-4 py-1 mb-4 border-l border-gray-700 relative" ]
                [ dropDownSelect labelDropdownConfig model.filterState.labelDropdownState
                ]
            , H.div [ HA.class "px-4 py-1 mb-4 border-l border-gray-700 relative" ]
                [ dropDownSelect stateDropdownConfig model.filterState.stateDropdownState
                ]
            , H.div [ HA.class "px-4 py-1 mb-4 border-l border-gray-700 relative" ]
                [ dropDownSelect ownerDropdownConfig model.filterState.ownerDropdownState
                ]
            , H.div [ HA.class "px-4 py-1 mb-4 border-l border-gray-700" ]
                [ H.label []
                    [ H.input
                        [ HA.class "mr-1"
                        , HA.type_ "checkbox"
                        , HA.checked model.filterState.showArchived
                        , HE.onCheck (GotFilterUpdate << ShowArchivedFilter)
                        ]
                        []
                    , H.text "Show archived"
                    ]
                ]
            , H.div [ HA.class "px-4 py-1 mb-4 border-l border-gray-700" ]
                [ H.span
                    [ HA.class "cursor-pointer text-blue-500 hover:text-blue-400"
                    , HE.onClick (GotFilterUpdate ResetFilters)
                    ]
                    [ H.text "reset filters" ]
                ]
            , H.div [ HA.class "px-4 py-1" ]
                []
            ]
        , H.div [ HA.class "w-full flex flex-wrap items-baseline batchActions", invisible ]
            [ H.input
                [ HA.class "mx-2"
                , HA.type_ "checkbox"
                , HE.onCheck ToggleSelectAll
                , HA.checked (allVisibleExperimentsSelected model)
                ]
                []
            , H.ul
                [ HA.class "horizontal-list" ]
                [ H.li []
                    [ confirmableBatchButton model
                        PauseBtn
                        "Are you sure you want to pause all the selected experiments?"
                        pausing
                        enablePause
                    ]
                , H.li []
                    [ confirmableBatchButton model
                        ArchiveBtn
                        "Are you sure you want to archive all the selected experiments?"
                        archiving
                        enableArchive
                    ]
                , H.li [] [ openTBOverSelectedButton openingTensorBoard enableOpenTB ]
                , H.li []
                    [ confirmableBatchButton model
                        KillBtn
                        "Are you sure you want to kill all the selected experiments?"
                        killing
                        enableKill
                    ]
                , H.li [] [ nSelectedHtml ]
                ]
            ]
        ]


batchButton : String -> String -> Msg -> Bool -> Bool -> H.Html Msg
batchButton label bgColor message isLoading isActive =
    Page.Common.buttonCreator
        { action = Page.Common.SendMsg message
        , bgColor = bgColor
        , fgColor = "white"
        , isActive = isActive
        , isPending = isLoading
        , style = Page.Common.TextOnly
        , text = label
        }


confirmableBatchButton : LoadedModel -> ButtonID -> String -> Bool -> Bool -> H.Html Msg
confirmableBatchButton model btnID prompt isLoading isActive =
    let
        btnModel =
            getBtnModel model btnID

        config =
            btnModel.config

        updatedConfig =
            { config | isActive = isActive, isPending = isLoading }
    in
    Button.view { btnModel | config = updatedConfig, promptText = prompt }
        (ButtonMsg btnID)


openTBOverSelectedButton : Bool -> Bool -> H.Html Msg
openTBOverSelectedButton =
    batchButton "Open TensorBoard" "orange" OpenTBOverSelectedExperiments


toggleSelectAll : Bool -> LoadedModel -> LoadedModel
toggleSelectAll val model =
    let
        updater e =
            if filterExperimentResults model e then
                { e | selected = val }

            else
                e

        newExperiments =
            Maybe.map (List.map updater) model.experiments

        newModel =
            { model | experiments = newExperiments }
    in
    newModel


filterExperimentByDescription : String -> Types.Experiment -> Bool
filterExperimentByDescription query experiment =
    let
        lowerQuery =
            String.toLower query

        lowerDescription =
            String.toLower experiment.description
    in
    String.contains lowerQuery lowerDescription


filterExperimentByState : LoadedModel -> Types.Experiment -> Bool
filterExperimentByState model =
    .state >> DS.selectedOrClear model.filterState.stateDropdownState


filterExperimentByOwner : LoadedModel -> Types.Experiment -> Bool
filterExperimentByOwner model =
    .owner >> DS.selectedOrClear model.filterState.ownerDropdownState


filterExperimentByLabel : LoadedModel -> Types.Experiment -> Bool
filterExperimentByLabel model =
    .labels >> Set.toList >> DS.anyOfOrClear model.filterState.labelDropdownState


filterExperimentByArchived : LoadedModel -> Types.Experiment -> Bool
filterExperimentByArchived model experiment =
    if model.filterState.showArchived then
        -- Show everything, not just archived.
        True

    else
        not experiment.archived


filterExperimentResults : LoadedModel -> TableExperiment -> Bool
filterExperimentResults model tableExp =
    case tableExp.experimentResult of
        Ok exp ->
            List.all identity
                [ filterExperimentByDescription model.filterState.descriptionFilter exp
                , filterExperimentByLabel model exp
                , filterExperimentByState model exp
                , filterExperimentByOwner model exp
                , filterExperimentByArchived model exp
                ]

        Err _ ->
            False


viewTableBody : LoadedModel -> Session -> H.Html Msg
viewTableBody model sess =
    let
        filteredExperiments =
            model.experiments
                |> Maybe.Extra.unwrap [] (List.filter (filterExperimentResults model))
    in
    H.div [ HA.class "p-2" ]
        [ Table.view
            (tableConfig sess model)
            model.tableSortState
            (Just model.numEntriesToShow)
            filteredExperiments
        ]


labelDropdownConfig : DropdownConfig String Msg
labelDropdownConfig =
    { toMsg = GotFilterUpdate << LabelFilter
    , orderBySelected = False
    , filtering = False
    , title = "Label"
    , filterText = "Filter by label"
    , elementToString = identity
    }


stateDropdownConfig : DropdownConfig Types.RunState Msg
stateDropdownConfig =
    { toMsg = GotFilterUpdate << RunStateFilter
    , orderBySelected = False
    , filtering = False
    , title = "State"
    , filterText = "Filter by state"
    , elementToString = runStateToString
    }


ownerDropdownConfig : DropdownConfig Types.User Msg
ownerDropdownConfig =
    { toMsg = GotFilterUpdate << OwnerFilter
    , orderBySelected = False
    , filtering = False
    , title = "User"
    , filterText = "Filter by user"
    , elementToString = .username
    }


ownerCol : Table.Column TableExperiment Msg
ownerCol =
    Table.veryCustomColumn
        { name = "Owner"
        , id = "owner"
        , viewData =
            \tableExp ->
                let
                    username =
                        tableExp.experimentResult
                            |> Result.map (.username << .owner)
                            |> Result.withDefault ""

                    attributes =
                        case tableExp.experimentResult of
                            Ok exp ->
                                [ Page.Common.onClickStopPropagation (OwnerClicked exp.owner)
                                , HA.class "p-2 cursor-pointer hover:underline"
                                ]

                            Err _ ->
                                []
                in
                { attributes = attributes
                , children = [ H.text username ]
                }
        , sorter =
            Table.increasingOrDecreasingBy
                (Result.withDefault "" << Result.map (.username << .owner) << .experimentResult)
        }


viewForkBtn : Types.Experiment -> H.Html Msg
viewForkBtn exp =
    Page.Common.buttonCreator
        { action = Page.Common.SendMsg (ForkExperiment exp)
        , bgColor = "blue"
        , fgColor = "white"
        , isActive = True
        , isPending = False
        , style = Page.Common.IconOnly "fas fa-code-branch"
        , text = "Fork Experiment"
        }


viewTensorBoardForExpBtn : Int -> H.Html Msg
viewTensorBoardForExpBtn expId =
    let
        tbConfig =
            Types.FromExperimentIds [ expId ]
    in
    Page.Common.buttonCreator
        { action =
            Page.Common.SendMsg
                (TensorBoard.AccessTensorBoard tbConfig
                    |> GotTensorBoardLaunchCycleMsg (Set.fromList [ expId ])
                )
        , bgColor = "blue"
        , fgColor = "white"
        , isActive = True
        , isPending = False
        , style = Page.Common.IconOnly "fas fa-chart-line"
        , text = "Open TensorBoard"
        }


actionsCol : Table.Column TableExperiment Msg
actionsCol =
    Table.veryCustomColumn
        { name = "Actions"
        , id = "actions"
        , viewData =
            \tableExp ->
                let
                    child =
                        case tableExp.experimentResult of
                            Ok exp ->
                                H.ul [ HA.class "horizontal-list" ]
                                    [ H.li []
                                        [ viewForkBtn exp
                                        ]
                                    , H.li []
                                        [ viewTensorBoardForExpBtn exp.id
                                        ]
                                    ]

                            Err _ ->
                                H.text ""
                in
                { attributes = [ HA.class "p-2", Page.Common.onClickStopPropagation NoOp ]
                , children = [ child ]
                }
        , sorter = Table.unsortable
        }


viewCheckbox : TableExperiment -> Table.HtmlDetails Msg
viewCheckbox tableExp =
    case tableExp.experimentResult of
        Ok e ->
            let
                handler =
                    HE.onCheck (TableCheckboxClicked e.id)
            in
            { attributes = [ Page.Common.onClickStopPropagation NoOp ]
            , children =
                [ H.input
                    [ HA.type_ "checkbox"
                    , handler
                    , HA.checked tableExp.selected
                    ]
                    []
                ]
            }

        Err _ ->
            { attributes = []
            , children = [ H.text "error" ]
            }


checkboxColumn : Table.Column TableExperiment Msg
checkboxColumn =
    Table.veryCustomColumn
        { name = ""
        , id = ""
        , viewData = viewCheckbox
        , sorter = Table.unsortable
        }


tableConfig : Session -> LoadedModel -> Table.Config TableExperiment Msg
tableConfig sess m =
    let
        actionClasses =
            HA.class "font-semibold text-blue-500 hover:text-blue-300 cursor-pointer"

        nEntries =
            m.experiments
                |> Maybe.withDefault []
                |> List.length

        showMore =
            if nEntries > m.numEntriesToShow then
                let
                    remainder =
                        min entriesShowIncrement (nEntries - m.numEntriesToShow)
                in
                Just
                    (H.span
                        [ actionClasses, HE.onClick (ShowMore remainder) ]
                        [ "Show " ++ String.fromInt remainder ++ " more" |> H.text ]
                    )

            else
                Nothing

        showLess =
            if m.numEntriesToShow > entriesShowIncrement then
                let
                    reduceCount =
                        let
                            remainder =
                                modBy entriesShowIncrement m.numEntriesToShow
                        in
                        if remainder /= 0 then
                            remainder

                        else
                            entriesShowIncrement
                in
                Just
                    (H.span
                        [ actionClasses, HE.onClick (ShowLess reduceCount) ]
                        [ "Show " ++ String.fromInt reduceCount ++ " fewer" |> H.text ]
                    )

            else
                Nothing

        showAll =
            if nEntries > m.numEntriesToShow then
                Just
                    (H.span
                        [ actionClasses, HE.onClick ShowAll ]
                        [ "Show all (" ++ String.fromInt nEntries ++ ")" |> H.text ]
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
        { toId = .experimentResult >> unpack .id .id >> String.fromInt
        , toMsg = NewTableState
        , columns =
            [ checkboxColumn
            , Table.veryCustomColumn
                { name = "ID"
                , id = "id"
                , viewData =
                    \tableExp ->
                        let
                            id =
                                tableExp.experimentResult |> unpack .id .id
                        in
                        { attributes = [ HA.class "p-2 hover:underline", Page.Common.onClickStopPropagation NoOp ]
                        , children = [ H.a [ HA.href <| Route.toString <| Route.ExperimentDetail id ] [ H.text <| String.fromInt id ] ]
                        }
                , sorter = Table.decreasingOrIncreasingBy (unpack .id .id << .experimentResult)
                }
            , limitedWidthStringColWithIcon "20rem" "Description" "description" (unpack .description .description << .experimentResult) (unpack .archived .archived << .experimentResult)
            , ownerCol
            , labelsCol NoOp labelWidgetConfig
            , percentCol "Progress" "progress" <| unwrap -1 (.progress >> Maybe.Extra.unwrap -1 ((*) 100)) << .experimentResult
            , runStateCol (.experimentResult >> Result.toMaybe >> Maybe.map .state)
            , datetimeCol sess.zone "Start Time" "start-time" <| unwrap -1 (.startTime >> Time.posixToMillis) << .experimentResult
            , datetimeCol sess.zone "End Time" "end-time" <| unwrap -1 (.endTime >> Maybe.Extra.unwrap -1 Time.posixToMillis) << .experimentResult
            , actionsCol
            ]
        , customizations =
            let
                rowAttrs exp =
                    [ HA.class "cursor-pointer hover:bg-orange-100"
                    , HE.on "click"
                        (D.map (SendOut << Comm.RouteRequested (Route.ExperimentDetail <| unpack .id .id exp.experimentResult))
                            (D.map2 (||) (D.field "ctrlKey" D.bool) (D.field "metaKey" D.bool))
                        )
                    ]
            in
            { tableCustomizations | rowAttrs = rowAttrs, tfoot = Just footer }
        }


extractOwnerList : List Types.ExperimentResult -> List Types.User
extractOwnerList results =
    List.filterMap (Result.Extra.unwrap Nothing (\r -> Just r.owner)) results
        |> EverySet.fromList
        |> EverySet.toList


extractLabels : List Types.ExperimentResult -> List String
extractLabels results =
    List.filterMap (Result.Extra.unwrap Nothing (\r -> Just r.labels)) results
        |> List.foldr Set.union Set.empty
        |> Set.toList


updateLabelWidgetState : LW.State -> LoadedModel -> LoadedModel
updateLabelWidgetState targetState model =
    let
        targetId =
            targetState.id

        mapper exp =
            let
                expId =
                    Result.Extra.unpack .id .id exp.experimentResult
            in
            if expId == targetId then
                { exp | labelWidgetState = targetState }

            else
                exp

        experiments =
            Maybe.map (List.map mapper) model.experiments
    in
    { model | experiments = experiments }


labelsCol : msg -> LW.Config msg -> Table.Column TableExperiment msg
labelsCol noop config =
    let
        viewData tableExp =
            { attributes = [ HA.class "p-2", Page.Common.onClickStopPropagation noop ]
            , children = viewExp tableExp
            }

        viewExp tableExp =
            case tableExp.experimentResult of
                Ok exp ->
                    [ LW.view config tableExp.labelWidgetState exp.labels ]

                Err _ ->
                    []
    in
    Table.veryCustomColumn
        { name = "Labels"
        , id = "labels"
        , viewData = viewData
        , sorter = Table.unsortable
        }
