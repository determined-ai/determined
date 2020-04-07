module Msg exposing (Msg(..))

import API
import Browser
import Communication as Comm
import Http
import Page.Cluster
import Page.CommandList
import Page.ExperimentDetail
import Page.ExperimentList
import Page.LogViewer
import Page.Login
import Page.Logout
import Page.NotebookList
import Page.ShellList
import Page.TensorBoardList
import Page.TrialDetail
import Time
import Types
import Url exposing (Url)


type Msg
    = NoOp
    | GotDeterminedInfo Types.DeterminedInfo
    | GotTimeZone Time.Zone
    | UrlRequested Browser.UrlRequest
    | UrlChanged Url
    | SlotsTick
    | GotSlots (List Types.Slot)
    | ToggleUserDropdownMenu Bool
      -- Authentication stuff.
    | ValidatedAuthentication Bool Url (Result Http.Error Types.SessionUser)
    | GotAuthenticationResponse Url (Result Http.Error ())
      -- Individual pages.
    | ClusterMsg Page.Cluster.Msg
    | CommandListMsg Page.CommandList.Msg
    | ExperimentDetailMsg Page.ExperimentDetail.Msg
    | ExperimentListMsg Page.ExperimentList.Msg
    | LoginMsg Page.Login.Msg
    | LogoutMsg Page.Logout.Msg
    | NotebookListMsg Page.NotebookList.Msg
    | ShellListMsg Page.ShellList.Msg
    | TensorBoardListMsg Page.TensorBoardList.Msg
    | TrialDetailMsg Page.TrialDetail.Msg
    | LogViewerMsg Page.LogViewer.Msg
    | ToastExpired Int
      -- Errors.
    | GotCriticalError Comm.SystemError
    | GotAPIError API.APIError
