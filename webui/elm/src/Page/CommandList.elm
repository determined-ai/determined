module Page.CommandList exposing
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
import Html as H
import OutMessage
import Page.GenericListPage as Base
import Route
import Session exposing (Session)
import Types


pageInfo : Base.PageInfo Msg () Types.Command () ()
pageInfo =
    { name = "Commands"
    , toMsg = GotBaseMsg
    , routeConstructor = Route.CommandList
    , poll = API.pollCommands
    , getLogsPath = \id -> [ "commands", id, "events" ]
    , columns =
        [ Base.IdColumn
        , Base.OwnerColumn
        , Base.DescriptionColumn
        , Base.StateColumn
        , Base.StartTimeColumn
        , Base.ActionsColumn []
        ]
    , kill = API.killCommand
    , getOwner = .owner
    , getRegisteredTime = .registeredTime
    , getId = .id
    , getState = .state
    , getDescription = .description
    , toInternalData = always ()
    , initInternalState = always ()
    , header = Nothing
    , singularName = "Command"
    , pluralName = "commands"
    }


type alias Model =
    Base.Model Msg () Types.Command () ()


type Msg
    = GotBaseMsg (Base.Msg Msg () Types.Command)


type OutMsg
    = NoOp


init : Maybe Model -> Route.CommandLikeListOptions -> ( Model, Cmd Msg )
init previousModel options =
    let
        ( model, cmd ) =
            Base.init pageInfo previousModel options
    in
    ( model, Cmd.map GotBaseMsg cmd )


update : Msg -> Model -> Session -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg model session =
    case msg of
        GotBaseMsg subMsg ->
            Base.update pageInfo session subMsg model
                |> OutMessage.mapOutMsg (Maybe.map (Comm.map (\(Base.SendMsg ()) -> NoOp)))


view : Model -> Session -> H.Html Msg
view model session =
    Base.view pageInfo model session


subscriptions : Model -> Sub Msg
subscriptions model =
    Base.subscriptions pageInfo model
