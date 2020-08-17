module Page.TensorBoardList exposing
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
import Components.CollapsibleList as CollapsibleList
import Components.Table as Table exposing (Status(..))
import Html as H
import Html.Attributes as HA exposing (class)
import Modules.TensorBoard as TensorBoard
import Page.Common
import Page.GenericListPage as Base
import Route
import Session exposing (Session)
import Types


type alias TableRecord =
    Base.TableRecord Msg CustomMsg Types.TensorBoard CollapsibleList.Model


type alias Model =
    Base.Model Msg CustomMsg Types.TensorBoard CollapsibleList.Model ()


type CustomMsg
    = CollapsibleListMsg String CollapsibleList.Msg
    | OpenTensorBoard Types.TensorBoard


type Msg
    = GotBaseMsg (Base.Msg Msg CustomMsg Types.TensorBoard)


type OutMsg
    = NoOp


tensorboardSourcesToShow : Int
tensorboardSourcesToShow =
    5


pageInfo : Base.PageInfo Msg CustomMsg Types.TensorBoard CollapsibleList.Model ()
pageInfo =
    let
        openButtonFactory : TableRecord -> Page.Common.ButtonConfig CustomMsg
        openButtonFactory tableRecord =
            Page.Common.openButtonConfig
                (Page.Common.SendMsg (OpenTensorBoard tableRecord.record))
                (pageInfo.getState tableRecord.record |> Page.Common.isCommandOpenable)
    in
    { name = "TensorBoards"
    , toMsg = GotBaseMsg
    , routeConstructor = Route.TensorBoardList
    , poll = API.pollTensorBoards
    , getLogsPath = \id -> [ "tensorboard", id, "events" ]
    , columns =
        [ Base.IdColumn
        , Base.OwnerColumn
        , Base.StateColumn
        , Base.CustomColumn sourceColConfig
        , Base.StartTimeColumn
        , Base.ActionsColumn [ openButtonFactory ]
        ]
    , kill = API.killTensorBoard
    , getOwner = .owner
    , getRegisteredTime = .registeredTime
    , getId = .id
    , getState = .state
    , getDescription = .description
    , toInternalData = always (CollapsibleList.init tensorboardSourcesToShow)
    , initInternalState = always ()
    , header = Nothing
    , singularName = "TensorBoard"
    , pluralName = "TensorBoards"
    }


init : Maybe Model -> Route.CommandLikeListOptions -> ( Model, Cmd Msg )
init previousModel options =
    let
        ( model, cmd ) =
            Base.init pageInfo previousModel options
    in
    ( model, Cmd.map GotBaseMsg cmd )


handleCustomMsg : CustomMsg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
handleCustomMsg msg model _ =
    case msg of
        CollapsibleListMsg id clMsg ->
            let
                updateFn tableRecord =
                    { tableRecord
                        | customData =
                            CollapsibleList.update clMsg tableRecord.customData
                    }
            in
            ( Base.updateRecordById
                pageInfo
                updateFn
                id
                model
            , Cmd.none
            , Nothing
            )

        OpenTensorBoard tsb ->
            ( model, TensorBoard.openTensorBoard tsb, Nothing )


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


view : Model -> Session -> H.Html Msg
view model session =
    Base.view pageInfo model session


subscriptions : Model -> Sub Msg
subscriptions model =
    Base.subscriptions pageInfo model


tensorboardSourceSorter : TableRecord -> ( String, List Int )
tensorboardSourceSorter tr =
    case ( tr.record.expIds, tr.record.trialIds ) of
        ( Just ((_ :: _) as expIds), Nothing ) ->
            ( "Experiment", List.sort expIds )

        ( Nothing, Just ((_ :: _) as trialIds) ) ->
            ( "Trial", List.sort trialIds )

        _ ->
            -- Currently we do not support TensorBoards with a mixed source.
            ( "Unknown", [] )


sourceToHtml : TableRecord -> H.Html CustomMsg
sourceToHtml tableRecord =
    let
        tb =
            tableRecord.record

        linkHtml tagger id =
            H.a
                [ tagger id
                    |> Route.toString
                    |> HA.href
                , HA.class "link"
                ]
                [ String.fromInt id |> H.text ]

        linksToHtml links =
            links
                |> CollapsibleList.view tableRecord.customData (CollapsibleListMsg tb.id)

        content =
            case ( tb.expIds, tb.trialIds ) of
                ( Just ((_ :: rest) as expIds), Nothing ) ->
                    H.text
                        (if List.isEmpty rest then
                            "Experiment "

                         else
                            "Experiments "
                        )
                        :: (List.sort expIds
                                |> List.map (linkHtml Route.ExperimentDetail)
                                |> linksToHtml
                           )
                        |> Page.Common.horizontalList

                ( Nothing, Just ((_ :: rest) as trialIds) ) ->
                    H.text
                        (if List.isEmpty rest then
                            "Trial "

                         else
                            "Trials "
                        )
                        :: (List.sort trialIds
                                |> List.map (linkHtml Route.TrialDetailReact)
                                |> linksToHtml
                           )
                        |> Page.Common.horizontalList

                _ ->
                    H.text "Unknown"
    in
    H.p [ HA.style "max-width" "15rem" ] [ content ]


viewSource : TableRecord -> Table.HtmlDetails CustomMsg
viewSource tableTsb =
    { children = [ sourceToHtml tableTsb ], attributes = [] }


sourceColConfig :
    { name : String
    , id : String
    , viewData : TableRecord -> Table.HtmlDetails CustomMsg
    , sorter : Table.Sorter TableRecord
    }
sourceColConfig =
    { name = "Source"
    , id = "source"
    , viewData = viewSource
    , sorter = Table.increasingOrDecreasingBy tensorboardSourceSorter
    }
