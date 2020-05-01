module API exposing
    ( APIError
    , RequestHandlers
    , archiveExperiment
    , buildUrl
    , cancelExperiment
    , createExperiment
    , decodeCommandState
    , decodeDeterminedInfo
    , decodeExperiment
    , decodeExperimentMinimal
    , decodeExperimentResult
    , decodeExperimentSummaries
    , decodeGitMetadata
    , decodeLabels
    , decodeMaybe
    , decodeResult
    , decodeRunState
    , decodeTrialSummary
    , decodeUser
    , decodeValidationHistory
    , downloadTrialLogs
    , fetchDeterminedInfo
    , getExperimentSummary
    , killCommand
    , killExperiment
    , killNotebook
    , killShell
    , killTensorBoard
    , launchNotebook
    , launchTensorBoard
    , parseRunState
    , patchExperiment
    , pauseExperiment
    , pollCommandLogs
    , pollCommandTypeLogs
    , pollCommands
    , pollExperimentSummary
    , pollExperiments
    , pollMasterLogs
    , pollNotebookLogs
    , pollNotebooks
    , pollShellLogs
    , pollShells
    , pollSlots
    , pollTensorBoardLogs
    , pollTensorBoards
    , pollTrialDetail
    , pollTrialLogs
    , trialDetailsPage
    , trialLogsPage
    )

import Communication as Comm
import Dict
import Http
import Iso8601
import Json.Decode as D
    exposing
        ( Decoder
        , andThen
        , bool
        , dict
        , fail
        , float
        , int
        , list
        , nullable
        , string
        , succeed
        , value
        )
import Json.Decode.Pipeline as DP
    exposing
        ( custom
        , hardcoded
        , optional
        , optionalAt
        , required
        , requiredAt
        )
import Json.Encode as E
import Maybe.Extra
import Set exposing (Set)
import Types
import Url.Builder as UB


type APIError
    = BadStatus Int
    | DecodeError String
    | BadUrl


{-| RequestHandlers contains message constructors that are used after a REST request is made.
-}
type alias RequestHandlers msg body =
    { onSuccess : body -> msg
    , onSystemError : Comm.SystemError -> msg
    , onAPIError : APIError -> msg
    }


{-| For development, this can be changed to `Just` of a string (e.g., `Just
"http://localhost:8081"`) to prefix all API request URLs with that string.
-}
apiBase : Maybe String
apiBase =
    Nothing


buildUrl : List String -> List UB.QueryParameter -> String
buildUrl parts params =
    case apiBase of
        Just api ->
            UB.crossOrigin api parts params

        Nothing ->
            UB.absolute parts params


decodeResult : String -> (a -> Result e b) -> Decoder a -> Decoder b
decodeResult err fn decoder =
    decodeMaybe err (fn >> Result.toMaybe) decoder


decodeMaybe : String -> (a -> Maybe b) -> Decoder a -> Decoder b
decodeMaybe err fn decoder =
    decoder |> andThen (fn >> Maybe.Extra.unwrap (fail err) succeed)


expectResponseHandler : (String -> Result x body) -> (x -> String) -> RequestHandlers msg body -> Result Http.Error String -> msg
expectResponseHandler mapper errorMapper requestHandlers response =
    case response of
        Err (Http.BadStatus 401) ->
            requestHandlers.onSystemError Comm.AuthenticationError

        Err (Http.BadStatus status) ->
            requestHandlers.onAPIError (BadStatus status)

        Err (Http.BadUrl _) ->
            requestHandlers.onAPIError BadUrl

        Err Http.Timeout ->
            requestHandlers.onSystemError Comm.Timeout

        Err Http.NetworkError ->
            requestHandlers.onSystemError Comm.NetworkDown

        Err (Http.BadBody x) ->
            requestHandlers.onAPIError (DecodeError x)

        Ok body ->
            case mapper body of
                Ok r ->
                    requestHandlers.onSuccess r

                Err e ->
                    requestHandlers.onAPIError (DecodeError (errorMapper e))


{-| buildExpectJson builds an Http.Expect to be used when a request is supposed to return valid
JSON.
-}
buildExpectJson : Decoder body -> RequestHandlers msg body -> Http.Expect msg
buildExpectJson decoder requestHandlers =
    let
        responseHandler =
            expectResponseHandler
                (D.decodeString decoder)
                D.errorToString
                requestHandlers
    in
    Http.expectString responseHandler


{-| buildExpectIgnore builds an Http.Expect to be used when the body of a response can be ignored.
-}
buildExpectIgnore : RequestHandlers msg () -> Http.Expect msg
buildExpectIgnore requestHandlers =
    let
        responseHandler =
            expectResponseHandler
                (Ok () |> always)
                (always "")
                requestHandlers
    in
    Http.expectString responseHandler


get : Http.Expect msg -> String -> Cmd msg
get expect url =
    Http.get
        { url = url
        , expect = expect
        }


post : Http.Expect msg -> Http.Body -> String -> Cmd msg
post expect body url =
    Http.post
        { url = url
        , body = body
        , expect = expect
        }


fetchDeterminedInfo : RequestHandlers msg Types.DeterminedInfo -> Cmd msg
fetchDeterminedInfo requestHandlers =
    buildUrl [ "info" ] []
        |> get (buildExpectJson decodeDeterminedInfo requestHandlers)


createExperiment : RequestHandlers msg Types.ExperimentDescriptor -> Types.ID -> String -> Cmd msg
createExperiment requestHandlers id rawYamlConfig =
    let
        request =
            E.object
                [ ( "experiment_config", E.string rawYamlConfig )
                , ( "parent_id", E.int id )
                ]
    in
    post
        (buildExpectJson decodeExperimentMinimal requestHandlers)
        (Http.jsonBody request)
        (buildUrl [ "experiments" ] [])


{-| XHR request for summary information about all experiments. For now,
we fetch summary information of all experiments and implement filtering
in the browser.
-}
pollExperiments : RequestHandlers msg (List Types.ExperimentResult) -> Cmd msg
pollExperiments requestHandlers =
    buildUrl [ "experiments" ] [ UB.string "filter" "all" ]
        |> get (buildExpectJson decodeExperimentSummaries requestHandlers)


getExperimentSummary : RequestHandlers msg Types.ExperimentResult -> Types.ID -> Cmd msg
getExperimentSummary requestHandlers id =
    buildUrl [ "experiments", String.fromInt id, "summary" ] []
        |> get (buildExpectJson decodeExperimentResult requestHandlers)


{-| XHR request for detailed information about an experiment.
-}
pollExperimentSummary : Types.ID -> RequestHandlers msg Types.ExperimentResult -> Cmd msg
pollExperimentSummary eid requestHandlers =
    buildUrl [ "experiments", String.fromInt eid, "summary" ] []
        |> get (buildExpectJson decodeExperimentResult requestHandlers)


downloadTrialLogs : Types.ID -> String
downloadTrialLogs trialId =
    buildUrl [ "trials", String.fromInt trialId, "logs" ] [ UB.string "format" "raw" ]


{-| XHR request for detailed information about a trial.
-}
pollTrialDetail : RequestHandlers msg Types.TrialDetail -> Types.ID -> Cmd msg
pollTrialDetail requestHandlers tid =
    buildUrl [ "trials", String.fromInt tid, "details" ] []
        |> get (buildExpectJson decodeTrialDetail requestHandlers)


trialDetailsPage : Types.ID -> String
trialDetailsPage trialId =
    buildUrl [ "ui", "trials", String.fromInt trialId ] []


{-| trialLogsPage constructs the url to the standalone trial logs viewer page.
-}
trialLogsPage : Types.ID -> String
trialLogsPage trialId =
    buildUrl [ "ui", "logs", "trials", String.fromInt trialId ] []


{-| XHR request for a master's log messages.
-}
pollMasterLogs :
    RequestHandlers msg (List Types.LogEntry)
    -> { greaterThanId : Maybe Int, lessThanId : Maybe Int, tailLimit : Maybe Int }
    -> Cmd msg
pollMasterLogs requestHandlers { greaterThanId, lessThanId, tailLimit } =
    let
        params =
            Maybe.Extra.values
                [ Maybe.map (UB.int "greater_than_id") greaterThanId
                , Maybe.map (UB.int "less_than_id") lessThanId
                , Maybe.map (UB.int "tail") tailLimit
                ]
    in
    buildUrl [ "logs" ] params
        |> get (buildExpectJson decodeTrialLogs requestHandlers)


{-| XHR request for a trial's log messages.
-}
pollTrialLogs :
    Types.ID
    -> RequestHandlers msg (List Types.LogEntry)
    -> { greaterThanId : Maybe Int, lessThanId : Maybe Int, tailLimit : Maybe Int }
    -> Cmd msg
pollTrialLogs tid requestHandlers { greaterThanId, lessThanId, tailLimit } =
    let
        params =
            Maybe.Extra.values
                [ Maybe.map (UB.int "greater_than_id") greaterThanId
                , Maybe.map (UB.int "less_than_id") lessThanId
                , Maybe.map (UB.int "tail") tailLimit
                ]
    in
    buildUrl [ "trials", String.fromInt tid, "logs" ] params
        |> get (buildExpectJson decodeTrialLogs requestHandlers)


patchExperiment : RequestHandlers msg () -> E.Value -> Types.ID -> Cmd msg
patchExperiment requestHandlers body id =
    Http.request
        { method = "PATCH"
        , headers = []
        , url = buildUrl [ "experiments", String.fromInt id ] []
        , body = Http.stringBody "application/merge-patch+json" (E.encode 0 body)
        , expect = buildExpectIgnore requestHandlers
        , timeout = Nothing
        , tracker = Nothing
        }


archiveExperiment : RequestHandlers msg () -> Bool -> Types.ID -> Cmd msg
archiveExperiment requestHandlers archived =
    patchExperiment requestHandlers (E.object [ ( "archived", E.bool archived ) ])


cancelExperiment : RequestHandlers msg () -> Types.ID -> Cmd msg
cancelExperiment requestHandlers =
    patchExperiment requestHandlers (E.object [ ( "state", E.string "STOPPING_CANCELED" ) ])


killExperiment : RequestHandlers msg () -> Types.ID -> Cmd msg
killExperiment requestHandlers id =
    post
        (buildExpectIgnore requestHandlers)
        Http.emptyBody
        (buildUrl [ "experiments", String.fromInt id, "kill" ] [])


pauseExperiment : RequestHandlers msg () -> Bool -> Types.ID -> Cmd msg
pauseExperiment requestHandlers paused =
    patchExperiment
        requestHandlers
        (E.object
            [ ( "state"
              , if paused then
                    E.string "PAUSED"

                else
                    E.string "ACTIVE"
              )
            ]
        )


decodeCommand : D.Decoder Types.Command
decodeCommand =
    D.succeed Types.Command
        |> DP.required "id" D.string
        |> DP.required "registered_time" Iso8601.decoder
        |> DP.required "state" decodeCommandState
        |> DP.requiredAt [ "config", "entrypoint" ] (D.list D.string)
        |> DP.required "owner" decodeUser
        |> DP.requiredAt [ "config", "description" ] D.string


decodeCommands : D.Decoder (List Types.Command)
decodeCommands =
    D.dict decodeCommand
        |> D.map Dict.values


pollCommands : RequestHandlers msg (List Types.Command) -> Cmd msg
pollCommands requestHandlers =
    buildUrl [ "commands" ] []
        |> get (buildExpectJson decodeCommands requestHandlers)


killCommand : RequestHandlers msg () -> String -> Cmd msg
killCommand requestHandlers id =
    Http.request
        { method = "DELETE"
        , headers = []
        , url = buildUrl [ "commands", id ] []
        , body = Http.emptyBody
        , expect = buildExpectIgnore requestHandlers
        , timeout = Nothing
        , tracker = Nothing
        }


decodeNotebook : D.Decoder Types.Notebook
decodeNotebook =
    D.succeed Types.Notebook
        |> DP.required "id" D.string
        |> DP.required "registered_time" Iso8601.decoder
        |> DP.required "state" decodeCommandState
        |> DP.required "owner" decodeUser
        |> DP.requiredAt [ "config", "description" ] D.string
        |> DP.required "service_address" D.string


decodeNotebooks : D.Decoder (List Types.Notebook)
decodeNotebooks =
    D.dict decodeNotebook
        |> D.map Dict.values


pollNotebooks : RequestHandlers msg (List Types.Notebook) -> Cmd msg
pollNotebooks requestHandlers =
    buildUrl [ "notebooks" ] []
        |> get (buildExpectJson decodeNotebooks requestHandlers)


decodeShell : D.Decoder Types.Shell
decodeShell =
    D.succeed Types.Shell
        |> DP.required "id" D.string
        |> DP.required "owner" decodeUser
        |> DP.required "state" decodeCommandState
        |> DP.requiredAt [ "config", "description" ] D.string
        |> DP.required "exit_status" (nullable D.string)
        |> DP.required "registered_time" Iso8601.decoder


decodeShells : D.Decoder (List Types.Shell)
decodeShells =
    D.keyValuePairs decodeShell
        |> D.map (List.unzip >> Tuple.second)


pollShells : RequestHandlers msg (List Types.Shell) -> Cmd msg
pollShells requestHandlers =
    buildUrl [ "shells" ] []
        |> get (buildExpectJson decodeShells requestHandlers)


launchNotebook : RequestHandlers msg Types.Notebook -> Types.NotebookLaunchConfig -> Cmd msg
launchNotebook requestHandlers launchConfig =
    let
        jsonObj =
            E.object
                [ ( "config", launchConfig.config )
                , ( "context", launchConfig.context )
                ]
    in
    post
        (buildExpectJson decodeNotebook requestHandlers)
        (Http.jsonBody jsonObj)
        (buildUrl [ "notebooks" ] [])


killNotebook : RequestHandlers msg () -> String -> Cmd msg
killNotebook requestHandlers id =
    Http.request
        { method = "DELETE"
        , headers = []
        , url = buildUrl [ "notebooks", id ] []
        , body = Http.emptyBody
        , expect = buildExpectIgnore requestHandlers
        , timeout = Nothing
        , tracker = Nothing
        }


killShell : RequestHandlers msg () -> String -> Cmd msg
killShell requestHandlers id =
    Http.request
        { method = "DELETE"
        , headers = []
        , url = buildUrl [ "shells", id ] []
        , body = Http.emptyBody
        , expect = buildExpectIgnore requestHandlers
        , timeout = Nothing
        , tracker = Nothing
        }


decodeTelemetry : Decoder Types.Telemetry
decodeTelemetry =
    D.succeed Types.Telemetry
        |> required "enabled" bool
        |> optional "segment_key" (D.map Just string) Nothing


decodeDeterminedInfo : Decoder Types.DeterminedInfo
decodeDeterminedInfo =
    D.succeed Types.DeterminedInfo
        |> required "cluster_id" string
        |> required "master_id" string
        |> required "telemetry" decodeTelemetry
        |> required "version" string


{-| Decode a Json list of experiment summaries.
-}
decodeExperimentSummaries : Decoder (List Types.ExperimentResult)
decodeExperimentSummaries =
    list decodeExperimentResult


decodeUser : Decoder Types.User
decodeUser =
    D.succeed Types.User
        |> required "username" string
        |> required "id" int


{-| Try decoding an experiment by first attempting the full decoding
logic, then backing up to minimal decoding logic if the previous fails. This
protects the entire application from crashing if there exists a bug in the
decoding logic, or if the response breaks some invariant in the decoding logic.
-}
decodeExperimentResult : Decoder Types.ExperimentResult
decodeExperimentResult =
    D.oneOf
        [ D.map Ok decodeExperiment
        , D.map Err decodeExperimentMinimal
        ]


{-| Decode the minimal set of experiment details required by the web UI to
identify an experiment. Currently, this is the experiment ID and
description.
-}
decodeExperimentMinimal : Decoder Types.ExperimentDescriptor
decodeExperimentMinimal =
    D.succeed Types.ExperimentDescriptor
        |> required "id" int
        |> required "archived" bool
        |> requiredAt [ "config", "description" ] string
        |> hardcoded Set.empty


{-| Decode a Json Experiment.
-}
decodeExperiment : Decoder Types.Experiment
decodeExperiment =
    D.at [ "config", "searcher", "metric" ] string
        |> andThen
            (\metric ->
                D.succeed Types.Experiment
                    |> required "id" int
                    |> requiredAt [ "config", "description" ] string
                    |> required "state" decodeRunState
                    |> required "archived" bool
                    |> required "config" (dict value)
                    |> required "progress" (nullable float)
                    |> required "start_time" Iso8601.decoder
                    |> required "end_time" (nullable Iso8601.decoder)
                    |> optional "validation_history"
                        (D.map Just (list decodeValidationHistory))
                        Nothing
                    |> optional "trials"
                        (D.map Just (list (decodeTrialSummary metric)))
                        Nothing
                    |> custom (D.maybe decodeGitMetadata)
                    |> optionalAt [ "config", "labels" ] decodeLabels Set.empty
                    |> optionalAt [ "config", "resources", "max_slots" ] (D.maybe int) Nothing
                    |> required "owner" decodeUser
            )


decodeTrialDetail : Decoder Types.TrialDetail
decodeTrialDetail =
    succeed Types.TrialDetail
        |> required "id" int
        |> required "experiment_id" int
        |> required "state" decodeRunState
        |> required "seed" int
        |> required "hparams" (dict D.value)
        |> required "start_time" Iso8601.decoder
        |> required "end_time" (nullable Iso8601.decoder)
        |> required "warm_start_checkpoint_id" (nullable int)
        |> required "steps" (list decodeStep)


decodeStep : Decoder Types.Step
decodeStep =
    succeed Types.Step
        |> required "id" int
        |> required "state" decodeRunState
        |> required "start_time" Iso8601.decoder
        |> required "end_time" (nullable Iso8601.decoder)
        |> required "avg_metrics" (nullable decodeMetrics)
        |> required "validation" (nullable decodeValidation)
        |> required "checkpoint" (nullable decodeCheckpoint)


decodeResources : Decoder Types.Resources
decodeResources =
    dict D.int


decodeCheckpoint : Decoder Types.Checkpoint
decodeCheckpoint =
    succeed Types.Checkpoint
        |> required "id" int
        |> required "step_id" int
        |> required "trial_id" int
        |> required "state" decodeCheckpointState
        |> required "start_time" Iso8601.decoder
        |> required "end_time" (nullable Iso8601.decoder)
        |> required "uuid" (nullable string)
        |> required "resources" (nullable decodeResources)
        |> required "validation_metric" (nullable float)


decodeValidation : Decoder Types.Validation
decodeValidation =
    succeed Types.Validation
        |> required "id" int
        |> required "state" decodeRunState
        |> required "start_time" Iso8601.decoder
        |> required "end_time" (nullable Iso8601.decoder)
        |> required "metrics" (nullable decodeValidationMetrics)


decodeRunState : Decoder Types.RunState
decodeRunState =
    decodeMaybe "invalid experiment state" parseRunState string


parseRunState : String -> Maybe Types.RunState
parseRunState s =
    case s of
        "ACTIVE" ->
            Just Types.Active

        "CANCELED" ->
            Just Types.Canceled

        "COMPLETED" ->
            Just Types.Completed

        "ERROR" ->
            Just Types.Error

        "PAUSED" ->
            Just Types.Paused

        "STOPPING_CANCELED" ->
            Just Types.StoppingCanceled

        "STOPPING_COMPLETED" ->
            Just Types.StoppingCompleted

        "STOPPING_ERROR" ->
            Just Types.StoppingError

        _ ->
            Nothing


decodeCheckpointState : Decoder Types.CheckpointState
decodeCheckpointState =
    decodeMaybe "invalid checkpoint state" parseCheckpointState string


parseCheckpointState : String -> Maybe Types.CheckpointState
parseCheckpointState s =
    case s of
        "ACTIVE" ->
            Just Types.CheckpointActive

        "COMPLETED" ->
            Just Types.CheckpointCompleted

        "ERROR" ->
            Just Types.CheckpointError

        "DELETED" ->
            Just Types.CheckpointDeleted

        _ ->
            Nothing


decodeValidationHistory : Decoder Types.ValidationHistory
decodeValidationHistory =
    D.succeed Types.ValidationHistory
        |> required "trial_id" int
        |> required "end_time" Iso8601.decoder
        |> optional "validation_error" (D.maybe float) Nothing


decodeGitMetadata : Decoder Types.GitMetadata
decodeGitMetadata =
    D.succeed Types.GitMetadata
        |> required "git_remote" D.string
        |> required "git_commit" D.string
        |> required "git_committer" D.string
        |> required "git_commit_date" Iso8601.decoder


{-| Decode a set of experiment labels. We receive the labels for an experiment
from the server as an object mapping label names to True, but we want to work
with the labels as a set.
-}
decodeLabels : Decoder (Set String)
decodeLabels =
    D.map Set.fromList (D.list D.string)


decodeTrialSummary : String -> Decoder Types.TrialSummary
decodeTrialSummary metric =
    D.succeed Types.TrialSummary
        |> required "id" int
        |> required "state" decodeRunState
        |> required "hparams" (dict D.value)
        |> required "start_time" Iso8601.decoder
        |> required "end_time" (nullable Iso8601.decoder)
        |> required "num_steps" int
        |> optionalAt [ "latest_validation_metrics", "validation_metrics", metric ]
            (D.map Just float)
            Nothing
        |> optional "best_validation_metric" (D.maybe float) Nothing
        |> required "best_available_checkpoint" (nullable decodeCheckpoint)


decodeValidationMetrics : Decoder Types.ValidationMetrics
decodeValidationMetrics =
    succeed identity |> required "validation_metrics" decodeMetrics


decodeMetrics : Decoder Types.Metrics
decodeMetrics =
    dict D.value


parseCommandState : String -> Maybe Types.CommandState
parseCommandState raw =
    case raw of
        "PENDING" ->
            Just Types.CmdPending

        "ASSIGNED" ->
            Just Types.CmdAssigned

        "PULLING" ->
            Just Types.CmdPulling

        "STARTING" ->
            Just Types.CmdStarting

        "RUNNING" ->
            Just Types.CmdRunning

        "TERMINATING" ->
            Just Types.CmdTerminating

        "TERMINATED" ->
            Just Types.CmdTerminated

        _ ->
            Nothing


decodeCommandState : D.Decoder Types.CommandState
decodeCommandState =
    decodeMaybe "invalid command state" parseCommandState D.string


decodeTrialLogs : Decoder (List Types.LogEntry)
decodeTrialLogs =
    list decodeLogEntry


decodeLogEntry : Decoder Types.LogEntry
decodeLogEntry =
    succeed Types.LogEntry
        |> required "id" int
        |> required "message"
            (D.map
                (\s ->
                    -- Old trial logs stripped trailing whitespace, but new ones include the output
                    -- verbatim.
                    if String.endsWith "\n" s then
                        s

                    else
                        s ++ "\n"
                )
                string
            )
        |> optional "level" (D.map Just string) Nothing
        |> optional "time" (D.map Just Iso8601.decoder) Nothing


{-| Decode a JSON list of slots from agents.

Input is a JSON object with agent UUIDs as keys and the corresponding agent data
as values. Output results in a list of slots for all agent objects.

-}
decodeSlotsFromAgents : Decoder (List Types.Slot)
decodeSlotsFromAgents =
    D.keyValuePairs decodeAgent
        |> D.map (List.map Tuple.second)
        |> D.map (List.concatMap .slots)


decodeAgent : Decoder Types.Agent
decodeAgent =
    D.succeed Types.Agent
        |> DP.required "id" D.string
        |> DP.required "registered_time" Iso8601.decoder
        |> DP.optional "slots" decodeSlots []


decodeSlots : Decoder (List Types.Slot)
decodeSlots =
    D.keyValuePairs decodeSlot
        |> D.map (List.map Tuple.second)


{-| Decode a JSON slot.
-}
decodeSlot : Decoder Types.Slot
decodeSlot =
    D.succeed Types.Slot
        |> DP.required "id" D.string
        |> DP.requiredAt [ "device", "type" ] decodeSlotType
        |> DP.requiredAt [ "device", "brand" ] D.string
        |> DP.optionalAt [ "container", "state" ] decodeSlotState Types.Free


decodeSlotState : Decoder Types.SlotState
decodeSlotState =
    decodeMaybe "invalid slot state" parseSlotState string


decodeSlotType : Decoder Types.SlotType
decodeSlotType =
    decodeMaybe "invalid slot type" parseSlotType string


{-| Parse a string into a `SlotState`.
-}
parseSlotState : String -> Maybe Types.SlotState
parseSlotState raw =
    case raw of
        "ASSIGNED" ->
            Just Types.Assigned

        "PULLING" ->
            Just Types.Pulling

        "STARTING" ->
            Just Types.Starting

        "RUNNING" ->
            Just Types.Running

        "TERMINATING" ->
            Just Types.Terminating

        "TERMINATED" ->
            Just Types.Terminated

        _ ->
            Nothing


parseSlotType : String -> Maybe Types.SlotType
parseSlotType type_ =
    case type_ of
        "gpu" ->
            Just Types.GPU

        "cpu" ->
            Just Types.CPU

        _ ->
            Nothing


{-| XHR request for agent slots.
-}
pollSlots : RequestHandlers msg (List Types.Slot) -> Cmd msg
pollSlots requestHandlers =
    buildUrl [ "agents" ] []
        |> get (buildExpectJson decodeSlotsFromAgents requestHandlers)



---- TensorBoards.


decodeTensorBoard : D.Decoder Types.TensorBoard
decodeTensorBoard =
    D.succeed Types.TensorBoard
        |> DP.required "id" D.string
        |> DP.required "registered_time" Iso8601.decoder
        |> DP.required "state" decodeCommandState
        |> DP.optionalAt [ "misc", "experiment_ids" ] (D.map Just (D.list D.int)) Nothing
        |> DP.optionalAt [ "misc", "trial_ids" ] (D.map Just (D.list D.int)) Nothing
        |> DP.required "owner" decodeUser
        |> DP.requiredAt [ "config", "description" ] D.string
        |> DP.required "service_address" D.string


launchTensorBoard : RequestHandlers msg Types.TensorBoard -> Types.TensorBoardLaunchConfig -> Cmd msg
launchTensorBoard requestHandlers launchConfig =
    let
        jsonObj =
            case launchConfig of
                Types.FromExperimentIds expIds ->
                    E.object
                        [ ( "experiment_ids", E.list E.int expIds ) ]

                Types.FromTrialIds trialIds ->
                    E.object
                        [ ( "trial_ids", E.list E.int trialIds ) ]
    in
    post
        (buildExpectJson decodeTensorBoard requestHandlers)
        (Http.jsonBody jsonObj)
        (buildUrl [ "tensorboard" ] [])


killTensorBoard : RequestHandlers msg () -> String -> Cmd msg
killTensorBoard requestHandlers id =
    Http.request
        { method = "DELETE"
        , headers = []
        , url = buildUrl [ "tensorboard", id ] []
        , body = Http.emptyBody
        , expect = buildExpectIgnore requestHandlers
        , timeout = Nothing
        , tracker = Nothing
        }


decodeTensorBoards : D.Decoder (List Types.TensorBoard)
decodeTensorBoards =
    D.dict decodeTensorBoard
        |> D.map Dict.values


pollTensorBoards : RequestHandlers msg (List Types.TensorBoard) -> Cmd msg
pollTensorBoards requestHandlers =
    buildUrl [ "tensorboard" ] []
        |> get (buildExpectJson decodeTensorBoards requestHandlers)



-- Notebook/Command/TensorBoards.


decodeCommandEventDetail : D.Decoder Types.CommandEventDetail
decodeCommandEventDetail =
    let
        failIfNull =
            Maybe.withDefault (fail "") << Maybe.map succeed

        emptyEnumField =
            andThen failIfNull << nullable << succeed
    in
    D.oneOf
        [ D.field "scheduled_event" (emptyEnumField Types.ScheduledEvent)
        , D.field "assigned_event" (emptyEnumField Types.AssignedEvent)
        , D.field "container_started_event" (emptyEnumField Types.ContainerStartedEvent)
        , D.field "service_ready_event" (emptyEnumField Types.ServiceReadyEvent)
        , D.field "terminate_request_event" (emptyEnumField Types.TerminateRequestEvent)
        , D.field "exited_event" (D.map Types.ExitedEvent string)
        , D.field "log_event" (D.map Types.LogEvent string)
        ]


decodeCommandEvent : D.Decoder Types.CommandEvent
decodeCommandEvent =
    D.succeed Types.CommandEvent
        |> required "parent_id" string
        |> required "seq" int
        |> required "time" Iso8601.decoder
        |> requiredAt [ "snapshot", "config", "description" ] string
        |> custom decodeCommandEventDetail


pollCommandTypeLogs :
    (String -> List String)
    -> String
    -> RequestHandlers msg (List Types.CommandEvent)
    -> { greaterThanId : Maybe Int, lessThanId : Maybe Int, tailLimit : Maybe Int }
    -> Cmd msg
pollCommandTypeLogs idToUrlParts id requestHandlers { greaterThanId, lessThanId, tailLimit } =
    let
        params =
            Maybe.Extra.values
                [ Maybe.map (UB.int "greater_than_id") greaterThanId
                , Maybe.map (UB.int "less_than_id") lessThanId
                , Maybe.map (UB.int "tail") tailLimit
                ]
    in
    buildUrl (idToUrlParts id) params
        |> get (buildExpectJson (list decodeCommandEvent) requestHandlers)


pollCommandLogs :
    String
    -> RequestHandlers msg (List Types.CommandEvent)
    -> { greaterThanId : Maybe Int, lessThanId : Maybe Int, tailLimit : Maybe Int }
    -> Cmd msg
pollCommandLogs =
    pollCommandTypeLogs
        (\id -> [ "commands", id, "events" ])


pollNotebookLogs :
    String
    -> RequestHandlers msg (List Types.CommandEvent)
    -> { greaterThanId : Maybe Int, lessThanId : Maybe Int, tailLimit : Maybe Int }
    -> Cmd msg
pollNotebookLogs =
    pollCommandTypeLogs
        (\id -> [ "notebooks", id, "events" ])


pollTensorBoardLogs :
    String
    -> RequestHandlers msg (List Types.CommandEvent)
    -> { greaterThanId : Maybe Int, lessThanId : Maybe Int, tailLimit : Maybe Int }
    -> Cmd msg
pollTensorBoardLogs =
    pollCommandTypeLogs
        (\id -> [ "tensorboard", id, "events" ])


pollShellLogs :
    String
    -> RequestHandlers msg (List Types.CommandEvent)
    -> { greaterThanId : Maybe Int, lessThanId : Maybe Int, tailLimit : Maybe Int }
    -> Cmd msg
pollShellLogs =
    pollCommandTypeLogs
        (\id -> [ "shells", id, "events" ])
