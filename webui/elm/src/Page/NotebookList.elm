module Page.NotebookList exposing
    ( Model
    , Msg
    , OutMsg(..)
    , init
    , subscriptions
    , update
    , view
    )

import API
import Communication as Comm
import Components.DropdownButton as DropButton
import Html as H
import Json.Encode as E
import Page.Common
import Page.GenericListPage as Base
import Route
import Session exposing (Session)
import Types
import Url.Builder as UB


type alias TableRecord =
    Base.TableRecord Msg CustomMsg Types.Notebook ()


type alias Model =
    Base.Model Msg CustomMsg Types.Notebook () DropButton.Model


type CustomMsg
    = LaunchNotebookDropdownMsg (DropButton.Msg Msg)
    | NotebookLaunched Types.Notebook
    | DoLaunchNotebook Types.NotebookLaunchConfig
      -- Error.
    | GotCriticalError Comm.SystemError
    | GotAPIError API.APIError


type Msg
    = GotBaseMsg (Base.Msg Msg CustomMsg Types.Notebook)
    | GotCustomMsg CustomMsg


type OutMsg
    = NoOp


pageInfo : Base.PageInfo Msg CustomMsg Types.Notebook () DropButton.Model
pageInfo =
    let
        openButtonFactory : TableRecord -> Page.Common.ButtonConfig CustomMsg
        openButtonFactory tableRecord =
            Page.Common.openButtonConfig
                (Page.Common.SendMsg (NotebookLaunched tableRecord.record))
                (pageInfo.getState tableRecord.record |> Page.Common.isCommandOpenable)
    in
    { name = "Notebooks"
    , toMsg = GotBaseMsg
    , routeConstructor = Route.NotebookList
    , poll = API.pollNotebooks
    , getLogsPath = \id -> [ "notebooks", id, "events" ]
    , columns =
        [ Base.IdColumn
        , Base.OwnerColumn
        , Base.DescriptionColumn
        , Base.StateColumn
        , Base.StartTimeColumn
        , Base.ActionsColumn [ openButtonFactory ]
        ]
    , kill = API.killNotebook
    , getOwner = .owner
    , getRegisteredTime = .registeredTime
    , getId = .id
    , getState = .state
    , getDescription = .description
    , toInternalData = always ()
    , initInternalState = always DropButton.init
    , header = Just viewHeader
    , singularName = "Notebook"
    , pluralName = "notebooks"
    }


notebookLaunchConfig : Int -> Types.NotebookLaunchConfig
notebookLaunchConfig slots =
    { config =
        E.object
            [ ( "resources"
              , E.object
                    [ ( "slots", E.int slots ) ]
              )
            ]
    , context = E.null
    }


init : Maybe Model -> Route.CommandLikeListOptions -> ( Model, Cmd Msg )
init previousModel options =
    let
        ( model, cmd ) =
            Base.init pageInfo previousModel options
    in
    ( model, Cmd.map GotBaseMsg cmd )


requestHandlers : (body -> Msg) -> API.RequestHandlers Msg body
requestHandlers onSuccess =
    { onSuccess = onSuccess
    , onSystemError = GotCustomMsg << GotCriticalError
    , onAPIError = GotCustomMsg << GotAPIError
    }


launchNotebook : Types.NotebookLaunchConfig -> Cmd Msg
launchNotebook =
    API.launchNotebook
        (requestHandlers (GotCustomMsg << NotebookLaunched))


handleCustomMsg : CustomMsg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
handleCustomMsg msg model session =
    case msg of
        LaunchNotebookDropdownMsg dropMsg ->
            case Base.getCustomModelData model of
                Nothing ->
                    -- This is a weird state to be in because it implies the Base model
                    -- is still "Loading", which we shouldn't be executing this function at all.
                    ( model, Cmd.none, Nothing )

                Just customData ->
                    let
                        ( newDropdown, outMsgMaybe ) =
                            DropButton.update dropMsg customData

                        newModel =
                            Base.updateCustomModelData newDropdown model
                    in
                    case outMsgMaybe of
                        Nothing ->
                            ( newModel, Cmd.none, Nothing )

                        Just outMsg ->
                            interpretDropdownOutMsg outMsg newModel session

        NotebookLaunched ntb ->
            let
                cmd =
                    Cmd.batch
                        [ Base.pollData pageInfo
                            |> Cmd.map GotBaseMsg
                        , Page.Common.openWaitPage (notebookEventLink ntb.id) ntb.serviceAddress
                        ]
            in
            ( model, cmd, Nothing )

        DoLaunchNotebook launchConfig ->
            let
                cmd =
                    launchNotebook launchConfig
            in
            ( model, cmd, Nothing )

        GotCriticalError e ->
            ( model, Cmd.none, Comm.Error e |> Just )

        GotAPIError e ->
            let
                -- TODO(jgevirtz): Report error to user.
                _ =
                    Debug.log "Failed to perform API request" e
            in
            ( model, Cmd.none, Nothing )


handleGenericOutMsg :
    Session
    -> Maybe (Comm.OutMessage (Base.OutMsg CustomMsg))
    -> Model
    -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
handleGenericOutMsg =
    Base.handleGenericOutMsg handleCustomMsg


update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg model session =
    case msg of
        GotBaseMsg subMsg ->
            let
                ( m, cmd, outMsg ) =
                    Base.update
                        pageInfo
                        session
                        subMsg
                        model

                ( m2, cmd2, outMsg2 ) =
                    handleGenericOutMsg
                        session
                        outMsg
                        m
            in
            ( m2
            , Cmd.batch
                [ cmd
                , cmd2
                ]
            , outMsg2
            )

        GotCustomMsg cMsg ->
            handleCustomMsg cMsg model session


interpretDropdownOutMsg :
    DropButton.OutMsg Msg
    -> Model
    -> Session
    -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
interpretDropdownOutMsg outMsg model sess =
    case outMsg of
        DropButton.OptionSelected action ->
            update action model sess


viewHeader : List TableRecord -> DropButton.Model -> H.Html CustomMsg
viewHeader _ dropButtonModel =
    H.map LaunchNotebookDropdownMsg <|
        DropButton.view
            launchNotebookDropdownConfig
            dropButtonModel


view : Model -> Session -> H.Html Msg
view model session =
    Base.view pageInfo model session



{- Configuration for Launch Notebook dropdown. -}


launchNotebookDropdownConfig : DropButton.Config Msg
launchNotebookDropdownConfig =
    { primaryButton = launchNotebookOption
    , selectOptions = otherOptions
    , bgColor = "orange"
    , fgColor = "white"
    }


launchNotebookOption : DropButton.SelectOption Msg
launchNotebookOption =
    { action = GotCustomMsg << DoLaunchNotebook <| notebookLaunchConfig 1
    , label = "Launch new notebook"
    }


otherOptions : List (DropButton.SelectOption Msg)
otherOptions =
    [ { action = GotCustomMsg << DoLaunchNotebook <| notebookLaunchConfig 0
      , label = "Launch new CPU-only notebook"
      }
    ]


subscriptions : Model -> Sub Msg
subscriptions model =
    Base.subscriptions pageInfo model


notebookEventLink : String -> String
notebookEventLink id =
    UB.relative [ "notebooks", id, "events" ]
        [ UB.int "tail" 1 ]
