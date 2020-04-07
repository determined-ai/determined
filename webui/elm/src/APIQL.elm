module APIQL exposing
    ( experimentDetailQuery
    , experimentListQuery
    , experimentSelector
    , sendQuery
    , trialDetailQuery
    , trialLogsQuery
    )

import API
import Communication as Comm
import DetQL.Enum.Order_by
import DetQL.InputObject as Input
import DetQL.Object
import DetQL.Object.Checkpoints
import DetQL.Object.Experiments
import DetQL.Object.Steps
import DetQL.Object.Steps_aggregate
import DetQL.Object.Steps_aggregate_fields
import DetQL.Object.Trial_logs
import DetQL.Object.Trials
import DetQL.Object.Users
import DetQL.Object.Validation_metrics
import DetQL.Object.Validations
import DetQL.Query as Query
import Graphql.Http
import Graphql.Operation
import Graphql.OptionalArgument as OA
import Graphql.SelectionSet as SS
import Http
import Json.Decode as D
import Maybe.Extra
import Result.Extra
import Set exposing (Set)
import Types



---- General helpers.


graphQLEndpoint : String
graphQLEndpoint =
    API.buildUrl [ "graphql" ] []


sendQuery : API.RequestHandlers msg body -> SS.SelectionSet body Graphql.Operation.RootQuery -> Cmd msg
sendQuery requestHandlers =
    Graphql.Http.queryRequest graphQLEndpoint
        >> Graphql.Http.send
            (\resp0 ->
                let
                    resp =
                        resp0
                            |> Graphql.Http.withSimpleHttpError
                            |> Result.mapError
                                (\e ->
                                    case e of
                                        Graphql.Http.HttpError err ->
                                            err

                                        Graphql.Http.GraphqlError _ _ ->
                                            Http.BadBody "bad gql"
                                )
                in
                case resp of
                    Err (Http.BadStatus 401) ->
                        requestHandlers.onSystemError Comm.AuthenticationError

                    Err (Http.BadStatus status) ->
                        requestHandlers.onAPIError (API.BadStatus status)

                    Err (Http.BadUrl _) ->
                        requestHandlers.onAPIError API.BadUrl

                    Err Http.Timeout ->
                        requestHandlers.onSystemError Comm.Timeout

                    Err Http.NetworkError ->
                        requestHandlers.onSystemError Comm.NetworkDown

                    Err (Http.BadBody x) ->
                        requestHandlers.onAPIError (API.DecodeError x)

                    Ok body ->
                        requestHandlers.onSuccess body
            )


{-| The technique for emulating optional arguments in the GraphQL library is based on functions that
modify specific fields of records that are passed to them. Writing those out inline makes everything
extremely verbose; `set` streamlines things a lot (and `Setter` in turn streamlines the type
annotation of `set`, though it isn't used after that).
-}
type alias Setter a b =
    a -> b -> b


arg :
    { checkpoint : Setter a { b | checkpoint : OA.OptionalArgument a }
    , end_time : Setter a { b | end_time : OA.OptionalArgument a }
    , eq_ : Setter a { b | eq_ : OA.OptionalArgument a }
    , gt_ : Setter a { b | gt_ : OA.OptionalArgument a }
    , id : Setter a { b | id : OA.OptionalArgument a }
    , limit : Setter a { b | limit : OA.OptionalArgument a }
    , lt_ : Setter a { b | lt_ : OA.OptionalArgument a }
    , metric_values : Setter a { b | metric_values : OA.OptionalArgument a }
    , path : Setter a { b | path : OA.OptionalArgument a }
    , order_by : Setter a { b | order_by : OA.OptionalArgument a }
    , raw : Setter a { b | raw : OA.OptionalArgument a }
    , signed : Setter a { b | signed : OA.OptionalArgument a }
    , state : Setter a { b | state : OA.OptionalArgument a }
    , trial_id : Setter a { b | trial_id : OA.OptionalArgument a }
    , where_ : Setter a { b | where_ : OA.OptionalArgument a }
    }
arg =
    { checkpoint = \x y -> { y | checkpoint = OA.Present x }
    , end_time = \x y -> { y | end_time = OA.Present x }
    , eq_ = \x y -> { y | eq_ = OA.Present x }
    , gt_ = \x y -> { y | gt_ = OA.Present x }
    , id = \x y -> { y | id = OA.Present x }
    , limit = \x y -> { y | limit = OA.Present x }
    , lt_ = \x y -> { y | lt_ = OA.Present x }
    , metric_values = \x y -> { y | metric_values = OA.Present x }
    , path = \x y -> { y | path = OA.Present x }
    , order_by = \x y -> { y | order_by = OA.Present x }
    , raw = \x y -> { y | raw = OA.Present x }
    , signed = \x y -> { y | signed = OA.Present x }
    , state = \x y -> { y | state = OA.Present x }
    , trial_id = \x y -> { y | trial_id = OA.Present x }
    , where_ = \x y -> { y | where_ = OA.Present x }
    }


mapDecode :
    D.Decoder decodesTo
    -> SS.SelectionSet D.Value typeLock
    -> SS.SelectionSet decodesTo typeLock
mapDecode decoder =
    SS.mapOrFail (D.decodeValue decoder >> Result.mapError D.errorToString)


mapDecodeMaybe :
    D.Decoder decodesTo
    -> SS.SelectionSet (Maybe D.Value) typeLock
    -> SS.SelectionSet (Maybe decodesTo) typeLock
mapDecodeMaybe decoder =
    SS.mapOrFail
        (Maybe.Extra.unwrap (Ok Nothing)
            (D.decodeValue decoder >> Result.Extra.mapBoth D.errorToString Just)
        )


{-| Get the JSON object at the given path inside the experiment config object (`Nothing` for the
entire object) and decode it with the given decoder.
-}
experimentConfigField :
    D.Decoder body
    -> Maybe String
    -> SS.SelectionSet body DetQL.Object.Experiments
experimentConfigField decoder path =
    DetQL.Object.Experiments.config (Maybe.Extra.unwrap identity arg.path path)
        |> mapDecode decoder


{-| The same as `SelectionSet.nonNullOrFail`, except that, for clarity's sake, it takes an ID that
will show up in error messages.
-}
nonNullOrFail :
    String
    -> SS.SelectionSet (Maybe decodesTo) typeLock
    -> SS.SelectionSet decodesTo typeLock
nonNullOrFail id (SS.SelectionSet fields decoder) =
    decoder
        |> D.andThen
            (Maybe.Extra.unwrap
                (D.fail ("Expected non-null but got null, ID: " ++ id))
                D.succeed
            )
        |> SS.SelectionSet fields


withNonNull :
    String
    -> SS.SelectionSet (Maybe decodesTo) typeLock
    -> SS.SelectionSet (decodesTo -> a) typeLock
    -> SS.SelectionSet a typeLock
withNonNull id s =
    SS.with (nonNullOrFail id s)


ifQL :
    Bool
    -> SS.SelectionSet (Maybe a) typeLock
    -> (SS.SelectionSet (Maybe a -> b) typeLock -> SS.SelectionSet b typeLock)
ifQL cond s =
    if cond then
        SS.with s

    else
        SS.hardcoded Nothing


idAsc :
    (({ a | id : OA.OptionalArgument DetQL.Enum.Order_by.Order_by }
      -> { a | id : OA.OptionalArgument DetQL.Enum.Order_by.Order_by }
     )
     -> b
    )
    -> { c | order_by : OA.OptionalArgument (List b) }
    -> { c | order_by : OA.OptionalArgument (List b) }
idAsc builder =
    arg.order_by [ builder (arg.id DetQL.Enum.Order_by.Asc) ]


decodeLabels : D.Decoder (Set String)
decodeLabels =
    D.oneOf
        [ D.null Set.empty
        , D.map Set.fromList (D.list D.string)
        ]



---- Specific parts within GraphQL queries.


checkpointSelector : SS.SelectionSet Types.Checkpoint DetQL.Object.Checkpoints
checkpointSelector =
    SS.succeed Types.Checkpoint
        |> SS.with DetQL.Object.Checkpoints.id
        |> SS.with DetQL.Object.Checkpoints.step_id
        |> SS.with DetQL.Object.Checkpoints.trial_id
        |> SS.with DetQL.Object.Checkpoints.state
        |> SS.with DetQL.Object.Checkpoints.start_time
        |> SS.with DetQL.Object.Checkpoints.end_time
        |> SS.with DetQL.Object.Checkpoints.uuid
        |> SS.with
            (DetQL.Object.Checkpoints.resources identity
                |> mapDecodeMaybe (D.dict D.int)
            )
        |> SS.with
            (DetQL.Object.Checkpoints.validation
                (DetQL.Object.Validations.metric_values
                    DetQL.Object.Validation_metrics.raw
                )
                |> SS.map (Maybe.Extra.join >> Maybe.Extra.join)
            )


stepSelector : SS.SelectionSet Types.Step DetQL.Object.Steps
stepSelector =
    SS.succeed Types.Step
        |> SS.with DetQL.Object.Steps.id
        |> SS.with DetQL.Object.Steps.state
        |> SS.with DetQL.Object.Steps.start_time
        |> SS.with DetQL.Object.Steps.end_time
        |> SS.with
            (DetQL.Object.Steps.metrics (arg.path "avg_metrics")
                |> mapDecodeMaybe (D.dict D.value)
            )
        |> SS.with
            (DetQL.Object.Steps.validation
                (SS.succeed Types.Validation
                    |> SS.with DetQL.Object.Validations.id
                    |> SS.with DetQL.Object.Validations.state
                    |> SS.with DetQL.Object.Validations.start_time
                    |> SS.with DetQL.Object.Validations.end_time
                    |> SS.with
                        (DetQL.Object.Validations.metrics (arg.path "validation_metrics")
                            |> mapDecodeMaybe (D.dict D.value)
                        )
                )
            )
        |> SS.with
            (DetQL.Object.Steps.checkpoint
                checkpointSelector
            )


trialDetailSelector : SS.SelectionSet Types.TrialDetail DetQL.Object.Trials
trialDetailSelector =
    SS.succeed Types.TrialDetail
        |> SS.with DetQL.Object.Trials.id
        |> SS.with DetQL.Object.Trials.experiment_id
        |> SS.with DetQL.Object.Trials.state
        |> SS.with DetQL.Object.Trials.seed
        |> SS.with (DetQL.Object.Trials.hparams identity |> mapDecode (D.dict D.value))
        |> SS.with DetQL.Object.Trials.start_time
        |> SS.with DetQL.Object.Trials.end_time
        |> SS.with DetQL.Object.Trials.warm_start_checkpoint_id
        |> SS.with (DetQL.Object.Trials.steps (idAsc Input.buildSteps_order_by) stepSelector)


trialSelector : SS.SelectionSet Types.TrialSummary DetQL.Object.Trials
trialSelector =
    let
        validationMetricSelector =
            DetQL.Object.Validations.metric_values DetQL.Object.Validation_metrics.raw
                |> SS.map Maybe.Extra.join

        validationOrderByMetric =
            arg.signed DetQL.Enum.Order_by.Asc
                |> Input.buildValidation_metrics_order_by
                |> arg.metric_values

        firstValidation selector where_ order =
            let
                fullWhere =
                    where_
                        >> (arg.eq_ Types.Completed
                                |> Input.buildValidation_state_comparison_exp
                                |> arg.state
                           )
            in
            -- Order all completed validations for this trial by the given criterion, take the first
            -- one, and extract the metric value that it contains.
            DetQL.Object.Trials.validations
                (arg.limit 1
                    >> arg.order_by [ Input.buildValidations_order_by order ]
                    >> (fullWhere
                            |> Input.buildValidations_bool_exp
                            |> arg.where_
                       )
                )
                selector
                |> SS.map (List.filterMap identity >> List.head)
    in
    SS.succeed Types.TrialSummary
        |> SS.with DetQL.Object.Trials.id
        |> SS.with DetQL.Object.Trials.state
        |> SS.with (DetQL.Object.Trials.hparams identity |> mapDecode (D.dict D.value))
        |> SS.with DetQL.Object.Trials.start_time
        |> SS.with DetQL.Object.Trials.end_time
        |> SS.with
            (DetQL.Object.Trials.steps_aggregate identity
                (DetQL.Object.Steps_aggregate.aggregate
                    (DetQL.Object.Steps_aggregate_fields.count identity |> nonNullOrFail "steps 1")
                    |> nonNullOrFail "steps 2"
                )
            )
        -- Metric value from the latest completed validation.
        |> SS.with (firstValidation validationMetricSelector identity (arg.id DetQL.Enum.Order_by.Desc))
        -- Metric value from the best completed validation.
        |> SS.with
            (firstValidation validationMetricSelector
                identity
                validationOrderByMetric
            )
        -- Best available checkpoint. The checkpoint must be completed as well as the validation.
        |> SS.with
            (firstValidation (DetQL.Object.Validations.checkpoint checkpointSelector)
                (arg.eq_ Types.CheckpointCompleted
                    |> Input.buildCheckpoint_state_comparison_exp
                    |> arg.state
                    |> Input.buildCheckpoints_bool_exp
                    |> arg.checkpoint
                )
                validationOrderByMetric
            )


experimentSelector : Bool -> SS.SelectionSet Types.Experiment DetQL.Object.Experiments
experimentSelector detailed =
    SS.succeed Types.Experiment
        |> SS.with DetQL.Object.Experiments.id
        |> SS.with (experimentConfigField D.string (Just "description"))
        |> SS.with DetQL.Object.Experiments.state
        |> SS.with DetQL.Object.Experiments.archived
        |> SS.with (experimentConfigField (D.dict D.value) Nothing)
        |> SS.with DetQL.Object.Experiments.progress
        |> SS.with DetQL.Object.Experiments.start_time
        |> SS.with DetQL.Object.Experiments.end_time
        |> ifQL detailed
            (DetQL.Object.Experiments.best_validation_history
                (arg.raw DetQL.Enum.Order_by.Asc
                    |> Input.buildValidation_metrics_order_by
                    |> arg.metric_values
                    |> Input.buildValidations_order_by
                    |> List.singleton
                    |> arg.order_by
                )
                (SS.succeed Types.ValidationHistory
                    |> SS.with DetQL.Object.Validations.trial_id
                    |> withNonNull "best validation end time" DetQL.Object.Validations.end_time
                    |> withNonNull "best validation metric"
                        (DetQL.Object.Validations.metric_values
                            DetQL.Object.Validation_metrics.raw
                        )
                )
            )
        |> ifQL detailed
            (DetQL.Object.Experiments.trials
                (idAsc Input.buildTrials_order_by)
                trialSelector
                |> SS.map Just
            )
        |> SS.with
            (SS.succeed (Maybe.map4 Types.GitMetadata)
                |> SS.with DetQL.Object.Experiments.git_remote
                |> SS.with DetQL.Object.Experiments.git_commit
                |> SS.with DetQL.Object.Experiments.git_committer
                |> SS.with DetQL.Object.Experiments.git_commit_date
            )
        |> SS.with (experimentConfigField decodeLabels (Just "labels"))
        |> SS.with (experimentConfigField (D.maybe D.int) (Just "resources.max_slots"))
        |> SS.with
            (DetQL.Object.Experiments.owner
                (SS.succeed Types.User
                    |> SS.with DetQL.Object.Users.username
                    |> SS.with DetQL.Object.Users.id
                )
            )



---- Top-level root queries, ready to send out.


{-| The query for the experiment list view.
-}
experimentListQuery : SS.SelectionSet (List Types.Experiment) Graphql.Operation.RootQuery
experimentListQuery =
    Query.experiments (idAsc Input.buildExperiments_order_by) (experimentSelector False)


experimentDetailQuery :
    Types.ID
    -> SS.SelectionSet (Maybe Types.Experiment) Graphql.Operation.RootQuery
experimentDetailQuery id =
    Query.experiments_by_pk { id = id } (experimentSelector True)


trialDetailQuery : Types.ID -> SS.SelectionSet Types.TrialDetail Graphql.Operation.RootQuery
trialDetailQuery id =
    Query.trials_by_pk { id = id } trialDetailSelector |> nonNullOrFail "trial"


trialLogsQuery :
    Types.ID
    -> { greaterThanId : Maybe Int, lessThanId : Maybe Int, tailLimit : Maybe Int }
    -> SS.SelectionSet (List Types.LogEntry) Graphql.Operation.RootQuery
trialLogsQuery id { greaterThanId, lessThanId, tailLimit } =
    let
        trialIdCond =
            arg.eq_ id
                |> Input.buildInt_comparison_exp
                |> arg.trial_id

        idCond =
            (Maybe.Extra.unwrap identity arg.gt_ greaterThanId
                >> Maybe.Extra.unwrap identity arg.lt_ lessThanId
            )
                |> Input.buildInt_comparison_exp
                |> arg.id

        where_ =
            (trialIdCond >> idCond)
                |> Input.buildTrial_logs_bool_exp
                |> arg.where_

        ( order, reorderFunc ) =
            -- Due to the form of the GraphQL API, which mimics SQL, requesting a tail means we have
            -- to get the results in descending ID order and reverse them afterward. At least in Elm
            -- we can bake that in instead of having to remember to reverse the result separately.
            if Maybe.Extra.isJust tailLimit then
                ( DetQL.Enum.Order_by.Desc, List.reverse )

            else
                ( DetQL.Enum.Order_by.Asc, identity )

        orderBy =
            arg.order_by [ Input.buildTrial_logs_order_by (arg.id order) ]

        limit =
            Maybe.Extra.unwrap identity arg.limit tailLimit
    in
    Query.trial_logs
        (where_ >> orderBy >> limit)
        (SS.succeed Types.LogEntry
            |> SS.with DetQL.Object.Trial_logs.id
            |> SS.with DetQL.Object.Trial_logs.message
            |> SS.hardcoded Nothing
            |> SS.hardcoded Nothing
        )
        |> SS.map reorderFunc
