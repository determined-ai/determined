module Page.GenericListPage exposing
    ( Column(..)
    , Model
    , Msg
    , OutMsg(..)
    , PageInfo
    , TableRecord
    , getCustomModelData
    , handleGenericOutMsg
    , init
    , pollData
    , subscriptions
    , update
    , updateCustomModelData
    , updateRecordById
    , view
    )

import API
import Browser.Navigation as Navigation
import Communication as Comm
import Components.AdvancedButton as Button
import Components.DropdownSelect as DS
import Components.Logs as Logs
import Components.Table as Table
import Components.Table.Custom as Custom
import Dict
import EverySet
import Formatting
import Html as H
import Html.Attributes as HA
import List.Extra
import Maybe.Extra
import Modals.Logs as LogsModal
import Page.Common
import Route
import Session exposing (Session)
import Time exposing (Posix)
import Types
import Utils


{-| GenericListPage is an abstraction over fairly simple, list-based views which render a list of
objects into a `Components.Table`.

    The module relies heavily on type parameters in order to make the code generic enough to
    support a wide variety of list views, each of which might have slightly varying behavior.

    The generic types are:
      * msg: this is the message type used by the parent module. For instance,
        `CommandList.Msg`.
      * cMsg: short for "custom message". Custom messages arise from custom code supplied by the
        parent module. For instance, the `DropdownButton` in `NotebookList.elm`, which is
        defined as part of `NotebookList.pageInfo`, results in a
        `LaunchNotebookDropdownMsg`, which is a custom messge. Custom messages are routed
        to the parent module by way of the `GenericListPage.OutMsg`.
      * data: this is the data type that the list view is dealing with. For instance,
        `Types.Notebook`, `Types.Shell`, etc.
      * customRecordData: additional data associated with each piece of `data` (in other
        words, additional data for each possible row in the table).
      * customModelData: additional data stored in the model.

-}
type alias OptionalActions msg cMsg data customRecordData =
    List (TableRecord msg cMsg data customRecordData -> Page.Common.ButtonConfig cMsg)


type Column msg cMsg data customRecordData
    = IdColumn
    | OwnerColumn
    | DescriptionColumn
    | StateColumn
    | StartTimeColumn
    | ActionsColumn (OptionalActions msg cMsg data customRecordData)
    | CustomColumn
        { name : String
        , id : String
        , viewData : TableRecord msg cMsg data customRecordData -> Table.HtmlDetails cMsg
        , sorter : Table.Sorter (TableRecord msg cMsg data customRecordData)
        }


type TableRecordInternalData msg cMsg data
    = TableRecordInternalData (Button.Model (Msg msg cMsg data))


type alias TableRecord msg cMsg data customRecordData =
    { record : data
    , customData : customRecordData
    , internal : TableRecordInternalData msg cMsg data
    }


type alias FilterState =
    { stateDropdownState : DS.DropdownState Types.CommandState
    , ownerDropdownState : DS.DropdownState Types.User
    }


type alias LoadedModel msg cMsg data customRecordData customModelData =
    { data : List (TableRecord msg cMsg data customRecordData)
    , tableState : Table.State
    , filterState : FilterState
    , customData : customModelData
    , logsModal : LogsModal.Model
    }


type Model msg cMsg data customRecordData customModelData
    = Loading (Maybe FilterState) (Maybe Table.State) Route.CommandLikeListOptions
    | Loaded (LoadedModel msg cMsg data customRecordData customModelData)


type Msg msg cMsg data
    = CustomMsg cMsg
    | DoKillResource String
    | GotData (List data)
    | NewOwnerDropdownState (DS.DropdownState Types.User)
    | NewStateDropdownState (DS.DropdownState Types.CommandState)
    | NewTableState Table.State
    | ResourceKilled
    | Tick
      -- Logs.
    | LogsModalMsg LogsModal.Msg
    | OpenLogs data
      -- Errors.
    | GotAPIError API.APIError
    | GotCriticalError Comm.SystemError
      -- Buttons.
    | ButtonMsg String Button.Msg


type OutMsg cMsg
    = SendMsg cMsg


type alias PageInfo msg cMsg data customRecordData customModelData =
    { name : String
    , toMsg : Msg msg cMsg data -> msg
    , routeConstructor : Route.CommandLikeListOptions -> Route.Route
    , poll : API.RequestHandlers (Msg msg cMsg data) (List data) -> Cmd (Msg msg cMsg data)
    , getLogsPath : String -> List String
    , columns : List (Column msg cMsg data customRecordData)
    , kill : API.RequestHandlers (Msg msg cMsg data) () -> String -> Cmd (Msg msg cMsg data)
    , getOwner : data -> Types.User
    , getRegisteredTime : data -> Posix
    , getId : data -> String
    , getState : data -> Types.CommandState
    , getDescription : data -> String
    , toInternalData : data -> customRecordData
    , initInternalState : List data -> customModelData
    , header :
        Maybe (List (TableRecord msg cMsg data customRecordData) -> customModelData -> H.Html cMsg)

    -- String used to generate language about the data type in question.
    , singularName : String
    , pluralName : String
    }


stateColumnID : String
stateColumnID =
    "state"


commandStatesList : List Types.CommandState
commandStatesList =
    [ Types.CmdPending
    , Types.CmdAssigned
    , Types.CmdPulling
    , Types.CmdStarting
    , Types.CmdRunning
    , Types.CmdTerminating
    , Types.CmdTerminated
    ]


mapHtmlDetails : Table.HtmlDetails cMsg -> Table.HtmlDetails (Msg msg cMsg data)
mapHtmlDetails details =
    { attributes = List.map (HA.map CustomMsg) details.attributes
    , children = List.map (H.map CustomMsg) details.children
    }


viewState : PageInfo msg cMsg data customRecordData customModelData -> data -> Table.HtmlDetails cMsg
viewState pageInfo datum =
    let
        html =
            pageInfo.getState datum
                |> Page.Common.commandStateToSpan
    in
    { children = [ html ], attributes = [ HA.class "p-2" ] }


stateCol : PageInfo msg cMsg data customRecordData customModelData -> Table.Column (TableRecord msg cMsg data customRecordData) (Msg msg cMsg data)
stateCol pageInfo =
    Table.veryCustomColumn
        { id = stateColumnID
        , name = "State"
        , viewData = mapHtmlDetails << viewState pageInfo << .record
        , sorter =
            Table.increasingOrDecreasingBy
                (Custom.commandStateSorter << pageInfo.getState << .record)
        }


getInternalData :
    TableRecordInternalData msg cMsg data
    -> Button.Model (Msg msg cMsg data)
getInternalData (TableRecordInternalData button) =
    button


viewOps :
    PageInfo msg cMsg data customRecordData customModelData
    -> OptionalActions msg cMsg data customRecordData
    -> TableRecord msg cMsg data customRecordData
    -> Table.HtmlDetails (Msg msg cMsg data)
viewOps pageInfo optionalActions tableRecord =
    let
        killHandler =
            ButtonMsg (pageInfo.getId tableRecord.record)

        buttonDivider =
            H.span [ HA.class "mr-1" ] []

        commonActionsHtml =
            [ Button.view (getInternalData tableRecord.internal) killHandler
            , buttonDivider
            , Page.Common.openLogsButton (OpenLogs tableRecord.record)
            ]

        customActionsHtml =
            List.map (\oa -> oa tableRecord) optionalActions
                |> List.map (H.map CustomMsg << Page.Common.buttonCreator)
                |> List.intersperse buttonDivider

        actions =
            case customActionsHtml of
                [] ->
                    commonActionsHtml

                _ ->
                    customActionsHtml ++ buttonDivider :: commonActionsHtml
    in
    Table.HtmlDetails [ HA.class "p-2" ]
        [ H.div
            []
            actions
        ]


columnToTableColumn :
    PageInfo msg cMsg data customRecordData customModelData
    -> Session
    -> Column msg cMsg data customRecordData
    -> Table.Column (TableRecord msg cMsg data customRecordData) (Msg msg cMsg data)
columnToTableColumn pageInfo session column =
    case column of
        IdColumn ->
            Custom.stringCol "ID" "id" (pageInfo.getId << .record)

        OwnerColumn ->
            Custom.stringCol "Owner" "owner" (.username << pageInfo.getOwner << .record)

        DescriptionColumn ->
            Custom.stringCol "Description" "description" (pageInfo.getDescription << .record)

        StateColumn ->
            stateCol pageInfo

        StartTimeColumn ->
            Custom.datetimeCol
                session.zone
                "Start Time"
                "start-time"
                (Time.posixToMillis
                    << pageInfo.getRegisteredTime
                    << .record
                )

        ActionsColumn optionalActions ->
            Custom.htmlUnsortableColumn "Actions" "actions" (viewOps pageInfo optionalActions)

        CustomColumn columnConfig ->
            let
                mappedColumnConfig =
                    { name = columnConfig.name
                    , id = columnConfig.id
                    , viewData = mapHtmlDetails << columnConfig.viewData
                    , sorter = columnConfig.sorter
                    }
            in
            Table.veryCustomColumn mappedColumnConfig


tableConfig :
    PageInfo msg cMsg data customRecordData customModelData
    -> Session
    -> Table.Config (TableRecord msg cMsg data customRecordData) (Msg msg cMsg data)
tableConfig pageInfo session =
    Table.customConfig
        { toId = pageInfo.getId << .record
        , toMsg = NewTableState
        , columns =
            List.map (columnToTableColumn pageInfo session) pageInfo.columns
        , customizations = Custom.tableCustomizations
        }


stateDropdownConfig : DS.DropdownConfig Types.CommandState (Msg msg cMsg data)
stateDropdownConfig =
    { toMsg = NewStateDropdownState
    , orderBySelected = False
    , filtering = False
    , title = "State"
    , filterText = "Filter by state"
    , elementToString = Formatting.commandStateToString
    }


ownerDropdownConfig : DS.DropdownConfig Types.User (Msg msg cMsg data)
ownerDropdownConfig =
    { toMsg = NewOwnerDropdownState
    , orderBySelected = False
    , filtering = False
    , title = "User"
    , filterText = "Filter by user"
    , elementToString = .username
    }


requestHandlers : (body -> Msg msg cMsg data) -> API.RequestHandlers (Msg msg cMsg data) body
requestHandlers onSuccess =
    { onSuccess = onSuccess
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIError
    }


pollData : PageInfo msg cMsg data customRecordData customModelData -> Cmd (Msg msg cMsg data)
pollData pageInfo =
    pageInfo.poll (requestHandlers GotData)


killResources : PageInfo msg cMsg data customRecordData customModelData -> String -> Cmd (Msg msg cMsg data)
killResources pageInfo =
    pageInfo.kill (requestHandlers (always ResourceKilled))


extractOwnerList :
    PageInfo msg cMsg data customRecordData customModelData
    -> List data
    -> List Types.User
extractOwnerList pageInfo results =
    List.map pageInfo.getOwner results
        |> EverySet.fromList
        |> EverySet.toList


recordToTableElement :
    PageInfo msg cMsg data customRecordData customModelData
    -> data
    -> TableRecord msg cMsg data customRecordData
recordToTableElement pageInfo record =
    { record = record
    , customData =
        pageInfo.toInternalData record
    , internal =
        TableRecordInternalData <|
            Button.init True
                ("Are you sure you want to kill this " ++ pageInfo.singularName ++ "?")
                (Page.Common.killButtonConfig
                    (pageInfo.getId record
                        |> DoKillResource
                    )
                    (pageInfo.getState record /= Types.CmdTerminated)
                )
    }


defaultFilterState :
    PageInfo msg cMsg data customRecordData customModelData
    -> Session
    -> List data
    -> FilterState
defaultFilterState pageInfo _ data =
    let
        owners =
            extractOwnerList pageInfo data
    in
    { stateDropdownState =
        DS.defaultInitialState commandStatesList
    , ownerDropdownState =
        DS.defaultInitialState owners
    }


initFilterState :
    PageInfo msg cMsg data customRecordData customModelData
    -> Session
    -> Maybe FilterState
    -> Route.CommandLikeListOptions
    -> List data
    -> FilterState
initFilterState pageInfo session filterState options data =
    let
        owners =
            extractOwnerList pageInfo data

        default =
            Maybe.withDefault
                (defaultFilterState pageInfo session data)
                filterState

        updateState : List a -> DS.DropdownState a -> DS.DropdownState a
        updateState selected =
            DS.clear >> DS.selectItems (EverySet.fromList selected)
    in
    { stateDropdownState =
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


initTableState : Maybe Table.State -> Route.CommandLikeListOptions -> Table.State
initTableState tableState options =
    let
        sortReversed =
            Maybe.withDefault
                False
                options.sortReversed

        default =
            Maybe.withDefault
                (Table.initialSortWithReverse stateColumnID sortReversed)
                tableState
    in
    Maybe.Extra.unwrap
        default
        (\col -> Table.initialSortWithReverse col sortReversed)
        options.sort


initLoaded :
    PageInfo msg cMsg data customRecordData customModelData
    -> Session
    -> FilterState
    -> Table.State
    -> List data
    -> ( LoadedModel msg cMsg data customRecordData customModelData, Cmd (Msg msg cMsg data) )
initLoaded pageInfo session filterState tableState data =
    let
        model =
            { data =
                List.map (recordToTableElement pageInfo) data
            , tableState = tableState
            , filterState = filterState
            , customData = pageInfo.initInternalState data
            , logsModal = LogsModal.closed
            }

        cmd =
            updateQueryParameters pageInfo session model
    in
    ( model, cmd )


init :
    PageInfo msg cMsg data customRecordData customModelData
    -> Maybe (Model msg cMsg data customRecordData customModelData)
    -> Route.CommandLikeListOptions
    -> ( Model msg cMsg data customRecordData customModelData, Cmd (Msg msg cMsg data) )
init pageInfo previousModel options =
    let
        ( filterState, tableState ) =
            case previousModel of
                Just (Loaded lm) ->
                    ( Just lm.filterState, Just lm.tableState )

                _ ->
                    ( Nothing, Nothing )
    in
    ( Loading filterState tableState options
    , pollData pageInfo
    )


updateFilterState :
    (FilterState -> FilterState)
    -> LoadedModel msg cMsg data customRecordData customModelData
    -> LoadedModel msg cMsg data customRecordData customModelData
updateFilterState updateFn model =
    { model | filterState = updateFn model.filterState }


getCustomModelData :
    Model msg cMsg data customRecordData customModelData
    -> Maybe customModelData
getCustomModelData model =
    case model of
        Loading _ _ _ ->
            Nothing

        Loaded lm ->
            Just lm.customData


updateTableRecords :
    PageInfo msg cMsg data customRecordData customModelData
    -> List (TableRecord msg cMsg data customRecordData)
    -> List data
    -> Maybe (TableRecord msg cMsg data customRecordData -> data -> TableRecord msg cMsg data customRecordData)
    -> List (TableRecord msg cMsg data customRecordData)
updateTableRecords pageInfo recordStates records updateFnMaybe =
    let
        existingRecordStates =
            List.map (\recState -> ( recState.record |> pageInfo.getId, recState )) recordStates
                |> Dict.fromList

        mapper : data -> TableRecord msg cMsg data customRecordData
        mapper datum =
            case Dict.get (pageInfo.getId datum) existingRecordStates of
                Just recordState ->
                    case updateFnMaybe of
                        Just updateFn ->
                            updateFn recordState datum

                        Nothing ->
                            { recordState | record = datum }

                Nothing ->
                    recordToTableElement pageInfo datum
    in
    List.map mapper records


updateCustomModelData : customModelData -> Model msg cMsg data customRecordData customModelData -> Model msg cMsg data customRecordData customModelData
updateCustomModelData customData model =
    case model of
        Loading a b c ->
            Loading a b c

        Loaded lm ->
            Loaded { lm | customData = customData }


logsModalConfig :
    PageInfo msg cMsg data customRecordData customModelData
    -> data
    -> LogsModal.Config
logsModalConfig pageInfo resource =
    { pollInterval = Logs.defaultPollInterval
    , poll =
        pageInfo.getId resource
            |> API.pollCommandTypeLogs pageInfo.getLogsPath
    , description = pageInfo.getDescription resource
    }


updateLoaded :
    PageInfo msg cMsg data customRecordData customModelData
    -> Session
    -> Msg msg cMsg data
    -> LoadedModel msg cMsg data customRecordData customModelData
    -> ( LoadedModel msg cMsg data customRecordData customModelData, Cmd (Msg msg cMsg data), Maybe (Comm.OutMessage (OutMsg cMsg)) )
updateLoaded pageInfo session msg model =
    case msg of
        CustomMsg cMsg ->
            ( model
            , Cmd.none
            , Comm.OutMessage (SendMsg cMsg)
                |> Just
            )

        DoKillResource id ->
            ( model, killResources pageInfo id, Nothing )

        GotData data ->
            let
                owners =
                    extractOwnerList pageInfo data

                ownerDropdownState =
                    DS.setOptions owners model.filterState.ownerDropdownState

                filterStateUpdateFn fs =
                    { fs | ownerDropdownState = ownerDropdownState }

                updateRS :
                    TableRecord msg cMsg data customRecordData
                    -> data
                    -> TableRecord msg cMsg data customRecordData
                updateRS recordState record =
                    let
                        oldKillBtn =
                            getInternalData recordState.internal

                        oldKillBtnConfig =
                            oldKillBtn.config

                        updatedConfig =
                            { oldKillBtnConfig | isActive = pageInfo.getState record /= Types.CmdTerminated }

                        updatedKillBtn =
                            { oldKillBtn | config = updatedConfig }
                    in
                    { recordState
                        | record = record
                        , internal = TableRecordInternalData updatedKillBtn
                    }

                updatedRecords =
                    updateTableRecords pageInfo model.data data (Just updateRS)
                        |> List.sortBy
                            ((*) -1
                                << Time.posixToMillis
                                << pageInfo.getRegisteredTime
                                << .record
                            )

                updatedModel =
                    { model | data = updatedRecords }
            in
            ( updatedModel
                |> updateFilterState filterStateUpdateFn
            , Cmd.none
            , Nothing
            )

        NewOwnerDropdownState state ->
            let
                newModel =
                    updateFilterState (\fs -> { fs | ownerDropdownState = state }) model
            in
            ( newModel
            , updateQueryParameters pageInfo session newModel
            , Nothing
            )

        NewStateDropdownState state ->
            let
                newModel =
                    updateFilterState (\fs -> { fs | stateDropdownState = state }) model
            in
            ( newModel
            , updateQueryParameters pageInfo session newModel
            , Nothing
            )

        NewTableState state ->
            let
                newModel =
                    { model | tableState = state }
            in
            ( newModel
            , updateQueryParameters pageInfo session newModel
            , Nothing
            )

        ResourceKilled ->
            ( model, pollData pageInfo, Nothing )

        Tick ->
            ( model, pollData pageInfo, Nothing )

        LogsModalMsg modalMsg ->
            let
                ( m, c, e ) =
                    LogsModal.update
                        modalMsg
                        model.logsModal
            in
            ( { model | logsModal = m }
            , Cmd.map LogsModalMsg c
            , Maybe.map Comm.Error e
            )

        OpenLogs resource ->
            let
                modalConfig =
                    logsModalConfig pageInfo resource

                ( modalModel, modalCmd ) =
                    LogsModal.open modalConfig

                newModel =
                    { model | logsModal = modalModel }
            in
            ( newModel
            , Cmd.map LogsModalMsg modalCmd
            , Nothing
            )

        GotAPIError e ->
            let
                _ =
                    -- TODO(jgevirtz): Report error to user.
                    Debug.log "Failed to get or kill" e
            in
            ( model, Cmd.none, Nothing )

        GotCriticalError error ->
            ( model, Cmd.none, Comm.Error error |> Just )

        ButtonMsg recID btnMsg ->
            let
                maybeTableRecord =
                    List.Extra.find
                        ((==) recID << pageInfo.getId << .record)
                        model.data
            in
            case maybeTableRecord of
                Just tableRecord ->
                    let
                        ( newBtnModel, cmd ) =
                            Button.update
                                btnMsg
                                (getInternalData tableRecord.internal)

                        updatedTableRecord =
                            List.Extra.setIf
                                ((==) recID << pageInfo.getId << .record)
                                { tableRecord
                                    | internal = TableRecordInternalData newBtnModel
                                }
                                model.data
                    in
                    ( { model | data = updatedTableRecord }, cmd, Nothing )

                Nothing ->
                    ( model, Cmd.none, Nothing )


handleGenericOutMsg :
    (cMsg
     -> Model msg cMsg data customRecordData customModelData
     -> Session
     -> ( Model msg cMsg data customRecordData customModelData, Cmd msg, Maybe (Comm.OutMessage outMsg) )
    )
    -> Session
    -> Maybe (Comm.OutMessage (OutMsg cMsg))
    -> Model msg cMsg data customRecordData customModelData
    -> ( Model msg cMsg data customRecordData customModelData, Cmd msg, Maybe (Comm.OutMessage outMsg) )
handleGenericOutMsg customUpdate session maybeOutMsg model =
    case maybeOutMsg of
        Just (Comm.Error sysErr) ->
            ( model, Cmd.none, Comm.Error sysErr |> Just )

        Just (Comm.RaiseToast s) ->
            ( model, Cmd.none, Comm.RaiseToast s |> Just )

        Just (Comm.RouteRequested r b) ->
            ( model, Cmd.none, Comm.RouteRequested r b |> Just )

        Just (Comm.OutMessage (SendMsg sMsg)) ->
            customUpdate sMsg model session

        Nothing ->
            ( model, Cmd.none, Nothing )


update :
    PageInfo msg cMsg data customRecordData customModelData
    -> Session
    -> Msg msg cMsg data
    -> Model msg cMsg data customRecordData customModelData
    -> ( Model msg cMsg data customRecordData customModelData, Cmd msg, Maybe (Comm.OutMessage (OutMsg cMsg)) )
update pageInfo session msg model =
    case model of
        Loading filterState tableState options ->
            case msg of
                GotData data ->
                    let
                        initialTableState =
                            initTableState tableState options

                        initialFilterState =
                            initFilterState pageInfo session filterState options data

                        ( m, cmd ) =
                            initLoaded
                                pageInfo
                                session
                                initialFilterState
                                initialTableState
                                data
                    in
                    ( Loaded m
                    , Cmd.map pageInfo.toMsg cmd
                    , Nothing
                    )

                _ ->
                    ( model, Cmd.none, Nothing )

        Loaded m ->
            let
                ( lm, cmd, outMsg ) =
                    updateLoaded pageInfo session msg m
            in
            ( Loaded lm, Cmd.map pageInfo.toMsg cmd, outMsg )


updateQueryParameters :
    PageInfo msg cMsg data customRecordData customModelData
    -> Session
    -> LoadedModel msg cMsg d id is
    -> Cmd (Msg msg cMsg d)
updateQueryParameters pageInfo session m =
    let
        ( sort, sortReversed ) =
            Table.getSortState m.tableState

        options =
            { users =
                m.filterState.ownerDropdownState.selectedFilters
                    |> EverySet.toList
                    |> List.map .username
                    |> Utils.listToMaybe
            , states =
                m.filterState.stateDropdownState.selectedFilters
                    |> EverySet.toList
                    |> Utils.listToMaybe
            , sort =
                if sort == stateColumnID then
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
    Route.toString (pageInfo.routeConstructor options)
        |> Navigation.replaceUrl session.key


updateRecordById :
    PageInfo msg cMsg data customRecordData customModelData
    -> (TableRecord msg cMsg data customRecordData -> TableRecord msg cMsg data customRecordData)
    -> String
    -> Model msg cMsg data customRecordData customModelData
    -> Model msg cMsg data customRecordData customModelData
updateRecordById pageInfo updateFn id model =
    case model of
        Loading _ _ _ ->
            model

        Loaded lm ->
            let
                mapper tableRecord =
                    if pageInfo.getId tableRecord.record == id then
                        updateFn tableRecord

                    else
                        tableRecord

                updatedData =
                    List.map mapper lm.data
            in
            Loaded { lm | data = updatedData }


subscriptions : PageInfo msg cMsg data customRecordData customModelData -> Model msg cMsg data customRecordData customModelData -> Sub msg
subscriptions pageInfo model =
    Sub.map pageInfo.toMsg <|
        case model of
            Loading _ _ _ ->
                Sub.none

            Loaded lm ->
                let
                    tick =
                        Time.every 5000 (\_ -> Tick)

                    logsModalSub =
                        LogsModal.subscriptions lm.logsModal
                            |> Sub.map LogsModalMsg
                in
                Sub.batch [ tick, logsModalSub ]


filterByState :
    PageInfo msg cMsg data customRecordData customModelData
    -> LoadedModel msg cMsg data customRecordData customModelData
    -> data
    -> Bool
filterByState pageInfo model =
    pageInfo.getState >> DS.selectedOrClear model.filterState.stateDropdownState


filterByOwner :
    PageInfo msg cMsg data customRecordData customModelData
    -> LoadedModel msg cMsg data customRecordData customModelData
    -> data
    -> Bool
filterByOwner pageInfo model =
    pageInfo.getOwner >> DS.selectedOrClear model.filterState.ownerDropdownState


filterDatum :
    PageInfo msg cMsg data customRecordData customModelData
    -> LoadedModel msg cMsg data customRecordData customModelData
    -> data
    -> Bool
filterDatum pageInfo model datum =
    List.all identity
        [ filterByState pageInfo model datum
        , filterByOwner pageInfo model datum
        ]


viewEmpty : PageInfo msg cMsg data customRecordData customModelData -> Bool -> H.Html (Msg msg cMsg data)
viewEmpty pageInfo becauseOfFilters =
    let
        message =
            if becauseOfFilters then
                "No " ++ pageInfo.pluralName ++ " matching the selected filters were found."

            else
                "No " ++ pageInfo.pluralName ++ " have been started yet."
    in
    H.div [ HA.class "flex flex-row justify-center text-gray-600 text-2xl message" ]
        [ H.text message ]


viewTableHead :
    PageInfo msg cMsg data customRecordData customModelData
    -> LoadedModel msg cMsg data customRecordData customModelData
    -> H.Html (Msg msg cMsg data)
viewTableHead pageInfo model =
    H.div [ HA.class "w-full text-sm text-gray-700 pb-5" ]
        [ Page.Common.pageHeader pageInfo.name
        , H.div [ HA.class "w-full flex flex-wrap items-baseline pb-8" ]
            [ H.div [ HA.class "px-4 py-1 mb-4 border-gray-700 relative" ]
                [ DS.dropDownSelect stateDropdownConfig model.filterState.stateDropdownState
                ]
            , H.div [ HA.class "px-4 py-1 mb-4 border-l border-gray-700 relative" ]
                [ DS.dropDownSelect ownerDropdownConfig model.filterState.ownerDropdownState
                ]
            , H.div [ HA.class "px-4 py-1" ]
                []
            ]
        , H.map CustomMsg <|
            Maybe.Extra.unwrap
                (H.text "")
                (\fn -> fn model.data model.customData)
                pageInfo.header
        ]


viewTableBody :
    PageInfo msg cMsg data customRecordData customModelData
    -> LoadedModel msg cMsg data customRecordData customModelData
    -> Session
    -> H.Html (Msg msg cMsg data)
viewTableBody pageInfo model session =
    let
        filtered =
            model.data
                |> List.filter (.record >> filterDatum pageInfo model)
    in
    Table.view (tableConfig pageInfo session) model.tableState Nothing filtered


view : PageInfo msg cMsg data customRecordData customModelData -> Model msg cMsg data customRecordData customModelData -> Session -> H.Html msg
view pageInfo model session =
    H.map pageInfo.toMsg <|
        case model of
            Loading _ _ _ ->
                Page.Common.centeredLoadingWidget

            Loaded m ->
                let
                    empty =
                        not (List.any (.record >> filterDatum pageInfo m) m.data)

                    becauseOfFilters =
                        not (List.isEmpty m.data)
                in
                H.div [ HA.class "table w-full text-sm p-4" ]
                    [ viewTableHead pageInfo m
                    , if empty then
                        viewEmpty pageInfo becauseOfFilters

                      else
                        viewTableBody pageInfo m session
                    , LogsModal.view m.logsModal
                        |> H.map LogsModalMsg
                    ]
