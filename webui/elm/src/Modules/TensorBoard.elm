module Modules.TensorBoard exposing
    ( RequestError(..)
    , TensorBoardLaunchCycleMsg(..)
    , experimentHasMetrics
    , handleTensorBoardLaunchCycleMsg
    , openOrLaunchExpTensorBoard
    , openTensorBoard
    , trialHasMetrics
    )

import API
import Communication as Comm
import List.Extra
import Page.Common
import Set
import Types
import Url.Builder as UB


type RequestError
    = Critical Comm.SystemError
    | API API.APIError


{-| Cycle messages used for opening existing TensorBoards or launching new ones.
-}
type TensorBoardLaunchCycleMsg
    = AccessTensorBoard Types.TensorBoardLaunchConfig
    | TensorBoardLaunched Types.TensorBoard
    | GotTensorBoards Types.TensorBoardLaunchConfig (List Types.TensorBoard)
      -- Errors.
    | GotAPIError API.APIError
    | GotCriticalError Comm.SystemError


requestHandlers :
    (TensorBoardLaunchCycleMsg -> msg)
    -> (body -> TensorBoardLaunchCycleMsg)
    -> API.RequestHandlers msg body
requestHandlers toMsg onSuccess =
    { onSuccess = toMsg << onSuccess
    , onSystemError = toMsg << GotCriticalError
    , onAPIError = toMsg << GotAPIError
    }



---- TensorBoard link calculating functions.


tensorboardEventLink : String -> String
tensorboardEventLink id =
    API.buildUrl [ "tensorboard", id, "events" ] [ UB.int "tail" 1 ]


openTensorBoard : Types.TensorBoard -> Cmd msg
openTensorBoard tb =
    Page.Common.openWaitPage
        (tensorboardEventLink tb.id)
        tb.serviceAddress


satisfiesLaunchConfig : Types.TensorBoardLaunchConfig -> Types.TensorBoard -> Bool
satisfiesLaunchConfig launchConfig tensorboard =
    -- This does not take the checkpoint that the TensorBoard was created from into account.
    let
        areEqualUnordered listA listB =
            Set.fromList listA == Set.fromList listB

        matches =
            case launchConfig of
                Types.FromTrialIds trialIds ->
                    case tensorboard.trialIds of
                        Nothing ->
                            False

                        Just ids ->
                            areEqualUnordered ids trialIds

                Types.FromExperimentIds expIds ->
                    case tensorboard.expIds of
                        Nothing ->
                            False

                        Just ids ->
                            areEqualUnordered ids expIds
    in
    (tensorboard.state /= Types.CmdTerminating)
        && (tensorboard.state /= Types.CmdTerminated)
        && matches


{-| Either open an existing TensorBoard or launch a new one if there isn't one already.
-}
openOrLaunchExpTensorBoard :
    Maybe (List Types.TensorBoard)
    -> Types.TensorBoardLaunchConfig
    -> (TensorBoardLaunchCycleMsg -> msg)
    -> ( Types.RequestStatus RequestError Bool, Cmd msg )
openOrLaunchExpTensorBoard tsbList launchConfig toMsg =
    let
        found =
            tsbList
                |> Maybe.andThen
                    (List.Extra.find (satisfiesLaunchConfig launchConfig))

        handlers =
            requestHandlers toMsg TensorBoardLaunched
    in
    case found of
        Nothing ->
            ( Types.RequestPending, API.launchTensorBoard handlers launchConfig )

        Just tsb ->
            ( Types.RequestSettled True, openTensorBoard tsb )


{-| Query the server about TensorBoards and pass them off to openOrLaunchExpTensorBoard.
-}
handleTensorBoardLaunchCycleMsg :
    TensorBoardLaunchCycleMsg
    -> (TensorBoardLaunchCycleMsg -> msg)
    -> ( Types.RequestStatus RequestError Bool, Cmd msg )
handleTensorBoardLaunchCycleMsg cycleMsg toMsg =
    let
        _ =
            Debug.log "cyclemsg" cycleMsg
    in
    case cycleMsg of
        TensorBoardLaunched tb ->
            ( Types.RequestSettled True, openTensorBoard tb )

        AccessTensorBoard launchConfig ->
            ( Types.RequestPending
            , API.pollTensorBoards <|
                requestHandlers toMsg (GotTensorBoards launchConfig)
            )

        GotTensorBoards launchConfig tbs ->
            openOrLaunchExpTensorBoard
                (Just tbs)
                launchConfig
                toMsg

        -- Errors.
        GotAPIError e ->
            ( Types.RequestFailed (API e), Cmd.none )

        GotCriticalError e ->
            ( Types.RequestFailed (Critical e), Cmd.none )



---- Helpers to determine if metrics are available to be plotted.


trialHasMetrics : Types.TrialDetail -> Bool
trialHasMetrics trial =
    List.any (\step -> step.state == Types.Completed) trial.steps


experimentHasMetrics : Types.Experiment -> Bool
experimentHasMetrics experiment =
    let
        hasACompletedTrial trials =
            List.any (\trial -> trial.state == Types.Completed) trials

        hasTrialWithOneCompletedStep trials =
            List.any (\trial -> trial.numSteps > 1) trials
    in
    case experiment.trials of
        Just trials ->
            hasACompletedTrial trials
                || hasTrialWithOneCompletedStep trials

        Nothing ->
            False
