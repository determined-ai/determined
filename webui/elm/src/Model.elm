module Model exposing
    ( Model
    , Page(..)
    )

import Page.Cluster
import Page.CommandList
import Page.ExperimentDetail
import Page.ExperimentList
import Page.LogViewer
import Page.NotebookList
import Page.ShellList
import Page.TensorBoardList
import Page.TrialDetail
import Session exposing (Session)
import Toast exposing (Toast)
import Types


type alias Model =
    { session : Session
    , info : Maybe Types.DeterminedInfo
    , page : Page
    , criticalError : Maybe String
    , toasts : List Toast
    , nextToastID : Int
    , slots : Maybe (List Types.Slot)
    , slotsRequestPending : Bool
    , userDropdownOpen : Bool
    , previousExperimentListModel : Maybe Page.ExperimentList.Model
    , previousCommandListModel : Maybe Page.CommandList.Model
    , previousNotebookListModel : Maybe Page.NotebookList.Model
    , previousShellListModel : Maybe Page.ShellList.Model
    , previousTensorBoardListModel : Maybe Page.TensorBoardList.Model
    , version : String
    }


type Page
    = Init
    | NotFound
      -- Pages with corresponding subpage modules.
    | Cluster Page.Cluster.Model
    | CommandList Page.CommandList.Model
    | ExperimentDetail Page.ExperimentDetail.Model
    | ExperimentList Page.ExperimentList.Model
    | NotebookList Page.NotebookList.Model
    | ShellList Page.ShellList.Model
    | TensorBoardList Page.TensorBoardList.Model
    | TrialDetail Page.TrialDetail.Model
    | LogViewer Page.LogViewer.Model
