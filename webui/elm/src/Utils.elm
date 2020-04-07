module Utils exposing
    ( batchesPerStep
    , checkpointStorageLocation
    , getDescriptionFromConfig
    , getSmallerIsBetter
    , getStepValidation
    , ifThenElse
    , listToMaybe
    , searcherValidationMetricName
    )

import Dict
import Json.Decode as D
import Types


ifThenElse : Bool -> a -> a -> a
ifThenElse conditional trueCase falseCase =
    if conditional then
        trueCase

    else
        falseCase


{-| Return a `None` if the given list is empty, otherwise return (Just list).
-}
listToMaybe : List a -> Maybe (List a)
listToMaybe list =
    case list of
        [] ->
            Nothing

        l ->
            Just l



---- Experiment configuration getters.


searcherValidationMetricName : Types.ExperimentConfig -> Maybe String
searcherValidationMetricName config =
    Dict.get "searcher" config
        |> Maybe.andThen
            (D.decodeValue (D.field "metric" D.string) >> Result.toMaybe)


batchesPerStep : Types.ExperimentConfig -> Int
batchesPerStep config =
    Dict.get "batches_per_step" config
        |> Maybe.andThen
            (D.decodeValue D.int >> Result.toMaybe)
        |> Maybe.withDefault 100


checkpointStorageLocation : Types.ExperimentConfig -> Maybe Types.Storage
checkpointStorageLocation config =
    let
        getStorageField name =
            Dict.get "checkpoint_storage" config
                |> Maybe.andThen
                    (D.decodeValue (D.field name D.string) >> Result.toMaybe)
    in
    getStorageField "type"
        |> Maybe.andThen
            (\aType ->
                case aType of
                    "gcs" ->
                        getStorageField "bucket"
                            |> Maybe.andThen
                                (Types.GcsStorage >> Just)

                    "s3" ->
                        getStorageField "bucket"
                            |> Maybe.andThen
                                (Types.S3Storage >> Just)

                    "shared_fs" ->
                        let
                            maybeHostPath =
                                getStorageField "host_path"

                            maybeStoragePath =
                                getStorageField "storage_path"
                        in
                        case ( maybeHostPath, maybeStoragePath ) of
                            ( Just hostPath, Just storagePath ) ->
                                Types.SharedFSStroge hostPath (Just storagePath)
                                    |> Just

                            ( Just hostPath, Nothing ) ->
                                Types.SharedFSStroge hostPath Nothing
                                    |> Just

                            _ ->
                                Nothing

                    _ ->
                        Nothing
            )


getSmallerIsBetter : Types.ExperimentConfig -> Bool
getSmallerIsBetter config =
    Dict.get "searcher" config
        |> Maybe.andThen (D.decodeValue (D.field "smaller_is_better" D.bool) >> Result.toMaybe)
        |> Maybe.withDefault True


getDescriptionFromConfig : Types.ExperimentConfig -> Maybe String
getDescriptionFromConfig config =
    Dict.get "description" config
        |> Maybe.andThen (Result.toMaybe << D.decodeValue D.string)



---- Trial getters.


getStepValidation : Types.Step -> Maybe Float
getStepValidation step =
    step.checkpoint
        |> Maybe.andThen .validationMetric
