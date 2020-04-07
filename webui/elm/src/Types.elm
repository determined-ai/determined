module Types exposing
    ( Agent
    , Checkpoint
    , CheckpointState(..)
    , Command
    , CommandEvent
    , CommandEventDetail(..)
    , CommandState(..)
    , DeterminedInfo
    , Experiment
    , ExperimentConfig
    , ExperimentDescriptor
    , ExperimentResult
    , GitMetadata
    , ID
    , LogEntry
    , Metrics
    , Notebook
    , NotebookLaunchConfig
    , RequestStatus(..)
    , Resources
    , RunState(..)
    , SessionUser
    , Shell
    , Slot
    , SlotState(..)
    , SlotType(..)
    , Step
    , Storage(..)
    , TensorBoard
    , TensorBoardLaunchConfig(..)
    , TrialDetail
    , TrialSummary
    , User
    , Validation
    , ValidationHistory
    , ValidationMetrics
    , commandStatesList
    )

import Dict exposing (Dict)
import Json.Decode as D
import Json.Encode as E
import Set exposing (Set)
import Time exposing (Posix)


{-| Unique identifier type for experiments, trials, steps.
-}
type alias ID =
    Int


{-| DeterminedInfo describes a Determined platform info.
-}
type alias DeterminedInfo =
    { clusterId : String
    , masterId : String
    , version : String
    }


{-| User describes a user in Determined.
-}
type alias User =
    { username : String
    , id : Int
    }


{-| SessionUser describes a user along with extra information about its privileges.
-}
type alias SessionUser =
    { user : User
    , admin : Bool
    , active : Bool
    }


{-| To gracefully handle errors in decoding logic, use Elm's Result
type to represent a decoding result. Successful decoding results in an
Experiment object, while a decoding "error" results in an ExperimentDescriptor
object. Note that the decoding error path still requires minimal decoding (id
and description). If the minimal decoding path fails as well as the main
decoding logic, the application crashes with a fatal error.
-}
type alias ExperimentResult =
    Result ExperimentDescriptor Experiment


{-| An experiment ID, along with a textual description and a set of labels.
-}
type alias ExperimentDescriptor =
    { id : ID
    , archived : Bool
    , description : String
    , labels : Set String
    }


{-| A full experiment with run state.

A more type-safe version of this would be a union type that separates out
different experiment states, since they don't share the same fields. See [make
impossible states
impossible](https://medium.com/elm-shorts/how-to-make-impossible-states-impossible-c12a07e907b5).
However there is so much overlap between the states that it ended up more work
than it was worth.

-}
type alias Experiment =
    { id : ID
    , description : String
    , state : RunState
    , archived : Bool
    , config : ExperimentConfig
    , progress : Maybe Float
    , startTime : Posix
    , endTime : Maybe Posix
    , validationHistory : Maybe (List ValidationHistory)
    , trials : Maybe (List TrialSummary)
    , gitMetadata : Maybe GitMetadata
    , labels : Set String
    , maxSlots : Maybe Int
    , owner : User
    }


type alias ExperimentConfig =
    Dict String D.Value


{-| Possible run states for experiments, trials, steps.
-}
type RunState
    = Active
    | Canceled
    | Completed
    | Error
    | Paused
    | StoppingCanceled
    | StoppingCompleted
    | StoppingError


{-| Possible states for checkpoints.
-}
type CheckpointState
    = CheckpointActive
    | CheckpointCompleted
    | CheckpointError
    | CheckpointDeleted


type RequestStatus error a
    = RequestPending
    | RequestSettled a
    | RequestFailed error


type alias ValidationHistory =
    { trialId : ID
    , endTime : Posix
    , validationError : Maybe Float
    }


type alias GitMetadata =
    { remote : String
    , commit : String
    , committer : String
    , commitDate : Posix
    }


{-| Data associated with a trial summary.

Trial summaries are returned as part of the per-experiment endpoint.
`TrialDetail` has more exhaustive information about a specific trial.

-}
type alias TrialSummary =
    { id : ID
    , state : RunState
    , hparams : Dict String E.Value
    , startTime : Posix
    , endTime : Maybe Posix
    , numSteps : Int
    , latestValidationMetric : Maybe Float
    , bestValidationMetric : Maybe Float
    , bestAvailableCheckpoint : Maybe Checkpoint
    }


{-| Detailed data associated with a trial.
-}
type alias TrialDetail =
    { id : ID
    , experimentId : ID
    , state : RunState
    , seed : Int
    , hparams : Dict String E.Value
    , startTime : Posix
    , endTime : Maybe Posix
    , warmStartCheckpointId : Maybe Int
    , steps : List Step
    }


type alias Step =
    { id : ID
    , state : RunState
    , startTime : Posix
    , endTime : Maybe Posix
    , averageMetrics : Maybe Metrics
    , validation : Maybe Validation
    , checkpoint : Maybe Checkpoint
    }


type alias Validation =
    { id : ID
    , state : RunState
    , startTime : Posix
    , endTime : Maybe Posix
    , metrics : Maybe ValidationMetrics
    }


{-| Checkpoint resources
-}
type alias Resources =
    Dict String Int


{-| Checkpoint storage types.
-}
type Storage
    = GcsStorage String
    | S3Storage String
    | SharedFSStroge String (Maybe String)


type alias Checkpoint =
    { id : ID
    , stepId : ID
    , trialId : ID
    , state : CheckpointState
    , startTime : Posix
    , endTime : Maybe Posix
    , uuid : Maybe String
    , resources : Maybe Resources
    , validationMetric : Maybe Float
    }


type alias ValidationMetrics =
    Metrics


type alias Metrics =
    Dict String E.Value


{-| A log entry.
-}
type alias LogEntry =
    { id : ID
    , message : String
    , level : Maybe String
    , time : Maybe Posix
    }


{-| An agent.
-}
type alias Agent =
    { uuid : String
    , registeredTime : Posix
    , slots : List Slot
    }


{-| A slot.
-}
type alias Slot =
    { id : String
    , slotType : SlotType
    , device : String
    , state : SlotState
    }


{-| Possible slot states.
-}
type SlotState
    = Free
      -- These are task states.
    | Assigned
    | Pulling
    | Starting
    | Running
    | Terminating
    | Terminated


type SlotType
    = GPU
    | CPU


{-| A command.
-}
type alias Command =
    { id : String
    , registeredTime : Posix
    , state : CommandState
    , entrypoint : List String
    , owner : User
    , description : String
    }


{-| Possible command states, prefixed with "Cmd" to avoid conflicts with other state types.
-}
type CommandState
    = CmdPending
    | CmdAssigned
    | CmdPulling
    | CmdStarting
    | CmdRunning
    | CmdTerminating
    | CmdTerminated


{-| A list of the possible command states is useful in various parts of the code.
-}
commandStatesList : List CommandState
commandStatesList =
    [ CmdPending
    , CmdAssigned
    , CmdPulling
    , CmdStarting
    , CmdRunning
    , CmdTerminating
    , CmdTerminated
    ]


{-| A notebook.
-}
type alias Notebook =
    { id : String
    , registeredTime : Posix
    , state : CommandState
    , owner : User
    , description : String
    , serviceAddress : String
    }


{-| A shell.
-}
type alias Shell =
    { id : String
    , owner : User
    , state : CommandState
    , description : String
    , exitStatus : Maybe String
    , registeredTime : Posix
    }


{-| A TensorBoard.
-}
type alias TensorBoard =
    { id : String
    , registeredTime : Posix
    , state : CommandState
    , expIds : Maybe (List Int)
    , trialIds : Maybe (List Int)
    , owner : User
    , description : String
    , serviceAddress : String
    }


{-| TensorBoard launch configuration.
-}
type TensorBoardLaunchConfig
    = FromTrialIds (List Int)
    | FromExperimentIds (List Int)


{-| Notebook launch configuration.
-}
type alias NotebookLaunchConfig =
    { config : E.Value
    , context : E.Value
    }



-- Types for Notebook/Command/TensorBoard logs.


type CommandEventDetail
    = ScheduledEvent
    | AssignedEvent
    | ContainerStartedEvent
    | ServiceReadyEvent
    | TerminateRequestEvent
    | ExitedEvent String
    | LogEvent String


type alias CommandEvent =
    { parentID : String
    , seq : Int
    , time : Posix
    , description : String
    , detail : CommandEventDetail
    }
