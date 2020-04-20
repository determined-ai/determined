module Route exposing
    ( CommandLikeListOptions
    , ExperimentListOptions
    , Route(..)
    , defaultCommandLikeListOptions
    , defaultExperimentListOptions
    , parse
    , toString
    )

import Dict
import Formatting
import Types
import Url exposing (Url)
import Url.Builder as UB exposing (absolute)
import Url.Parser as Parser exposing ((</>), (<?>), Parser, int, map, oneOf, s, top)
import Url.Parser.Query as Query


type alias ExperimentListOptions =
    { users : Maybe (List String)
    , labels : Maybe (List String)
    , states : Maybe (List Types.RunState)
    , description : Maybe String
    , archived : Maybe Bool
    , sort : Maybe String
    , sortReversed : Maybe Bool
    }


urlToPathString : Url.Url -> String
urlToPathString url =
    url.path
        ++ (case url.query of
                Just query ->
                    "?" ++ query

                Nothing ->
                    ""
           )
        ++ (case url.fragment of
                Just fragment ->
                    "#" ++ fragment

                Nothing ->
                    ""
           )


{-| CommandLikeListOptions is meant to be used for command, notebook, shell, and TensorBoard list
views since the corresponding records are so similar to one another.
-}
type alias CommandLikeListOptions =
    { users : Maybe (List String)
    , states : Maybe (List Types.CommandState)
    , sort : Maybe String
    , sortReversed : Maybe Bool
    }


defaultExperimentListOptions : ExperimentListOptions
defaultExperimentListOptions =
    { users = Nothing
    , labels = Nothing
    , states = Nothing
    , description = Nothing
    , archived = Nothing
    , sort = Nothing
    , sortReversed = Nothing
    }


defaultCommandLikeListOptions : CommandLikeListOptions
defaultCommandLikeListOptions =
    { users = Nothing
    , states = Nothing
    , sort = Nothing
    , sortReversed = Nothing
    }


runStateFromQueryParameter : String -> Maybe Types.RunState
runStateFromQueryParameter qp =
    case qp of
        "active" ->
            Just Types.Active

        "canceled" ->
            Just Types.Canceled

        "completed" ->
            Just Types.Completed

        "errored" ->
            Just Types.Error

        "paused" ->
            Just Types.Paused

        "stopping-canceled" ->
            Just Types.StoppingCanceled

        "stopping-completed" ->
            Just Types.StoppingCompleted

        "stopping-error" ->
            Just Types.StoppingError

        _ ->
            Nothing


commandStateFromQueryParameter : String -> Maybe Types.CommandState
commandStateFromQueryParameter qp =
    case qp of
        "pending" ->
            Just Types.CmdPending

        "assigned" ->
            Just Types.CmdAssigned

        "pulling" ->
            Just Types.CmdPulling

        "starting" ->
            Just Types.CmdStarting

        "running" ->
            Just Types.CmdRunning

        "terminating" ->
            Just Types.CmdTerminating

        "terminated" ->
            Just Types.CmdTerminated

        _ ->
            Nothing


type Route
    = Cluster
    | Dashboard
    | CommandList CommandLikeListOptions
    | ExperimentDetail Int
    | ExperimentList ExperimentListOptions
    | Login (Maybe Url.Url)
    | Logout
    | NotebookList CommandLikeListOptions
    | ShellList CommandLikeListOptions
    | TensorBoardList CommandLikeListOptions
    | TrialDetail Int
    | LogViewer Int


parser : Parser (Route -> a) a
parser =
    let
        truthValues =
            Dict.fromList
                [ ( "true", True )
                , ( "false", False )
                ]

        listMapper =
            Maybe.map (List.filter (not << String.isEmpty) << String.split ",")

        runStatesMapper =
            Maybe.map (List.filterMap runStateFromQueryParameter << String.split ",")

        commandStatesMapper =
            Maybe.map (List.filterMap commandStateFromQueryParameter << String.split ",")

        makeExpListRoute route =
            route
                <?> (Query.string "users" |> Query.map listMapper)
                <?> (Query.string "labels" |> Query.map listMapper)
                <?> (Query.string "states" |> Query.map runStatesMapper)
                <?> Query.string "description"
                <?> Query.enum "show-archived" truthValues
                <?> Query.string "sort"
                <?> Query.enum "sort-reversed" truthValues
                |> map ExperimentListOptions
                |> map ExperimentList

        commandLikeListRoute route constructor =
            route
                <?> (Query.string "users" |> Query.map listMapper)
                <?> (Query.string "states" |> Query.map commandStatesMapper)
                <?> Query.string "sort"
                <?> Query.enum "sort-reversed" truthValues
                |> map CommandLikeListOptions
                |> map constructor
    in
    oneOf
        [ map Cluster (s "ui" </> s "cluster")
        , commandLikeListRoute (s "ui" </> s "commands") CommandList
        , map Dashboard top
        , map ExperimentDetail (s "ui" </> s "experiments" </> int)
        , makeExpListRoute (s "ui" </> s "experiments")
        , makeExpListRoute (s "ui")
        , commandLikeListRoute (s "ui" </> s "notebooks") NotebookList
        , commandLikeListRoute (s "ui" </> s "shells") ShellList
        , commandLikeListRoute (s "ui" </> s "tensorboards") TensorBoardList
        , map TrialDetail (s "ui" </> s "trials" </> int)
        , map LogViewer (s "ui" </> s "logs" </> s "trials" </> int)
        ]


parse : Url -> Maybe Route
parse =
    Parser.parse parser


toString : Route -> String
toString r =
    let
        runStateToString =
            String.toLower << Formatting.runStateToString

        commandStateToString =
            String.toLower << Formatting.commandStateToString

        stringListToString =
            List.sort >> String.join ","

        makeParam tag mapper getter options =
            Maybe.map (UB.string tag << mapper) (getter options)

        commandStatesToString =
            List.map (commandStateToString >> String.toLower) >> stringListToString

        commandLikeParameters options =
            [ makeParam "users" stringListToString .users options
            , makeParam "states" commandStatesToString .states options
            , makeParam "sort" identity .sort options
            , makeParam "sort-reversed" Formatting.boolToString .sortReversed options
            ]
                |> List.filterMap identity
    in
    case r of
        Cluster ->
            absolute [ "ui", "cluster" ] []

        Dashboard ->
            absolute [ "det", "dashboard" ] []

        CommandList options ->
            absolute
                [ "ui", "commands" ]
                (commandLikeParameters options)

        ExperimentDetail id ->
            absolute [ "ui", "experiments", String.fromInt id ] []

        ExperimentList options ->
            let
                statesToString =
                    List.map (runStateToString >> String.toLower) >> stringListToString

                parameters =
                    [ makeParam "users" stringListToString .users options
                    , makeParam "labels" stringListToString .labels options
                    , makeParam "states" statesToString .states options
                    , makeParam "description" identity .description options
                    , makeParam "show-archived" Formatting.boolToString .archived options
                    , makeParam "sort" identity .sort options
                    , makeParam "sort-reversed" Formatting.boolToString .sortReversed options
                    ]
                        |> List.filterMap identity
            in
            absolute [ "ui", "experiments" ] parameters

        Login maybeRedirect ->
            case maybeRedirect of
                Just redirect ->
                    absolute
                        [ "det", "login" ]
                        [ urlToPathString redirect
                            |> UB.string "redirect"
                        ]

                Nothing ->
                    absolute [ "det", "login" ] []

        Logout ->
            absolute [ "det", "logout" ] []

        NotebookList options ->
            absolute
                [ "ui", "notebooks" ]
                (commandLikeParameters options)

        ShellList options ->
            absolute
                [ "ui", "shells" ]
                (commandLikeParameters options)

        TensorBoardList options ->
            absolute
                [ "ui", "tensorboards" ]
                (commandLikeParameters options)

        TrialDetail id ->
            absolute [ "ui", "trials", String.fromInt id ] []

        LogViewer id ->
            absolute [ "ui", "logs", "trials", String.fromInt id ] []
