module Modals.ShowCheckpoint exposing
    ( Model
    , Msg
    , init
    , openCheckpoint
    , subscriptions
    , update
    , view
    )

{-| A component for showing checkpoint info in a modal.
-}

import Browser.Events
import Dict
import Formatting
import Html as H
import Html.Attributes as HA
import Json.Decode as D
import Maybe.Extra
import Session exposing (Session)
import Time
import Types
import Utils
import View.Modal as Modal


type Model
    = Closed
    | OpenCheckpoint Types.Experiment Types.Checkpoint


type Msg
    = CloseModal


openCheckpoint : Types.Experiment -> Types.Checkpoint -> ( Model, Cmd Msg )
openCheckpoint experiment checkpoint =
    ( OpenCheckpoint experiment checkpoint, Cmd.none )


checkpointAddress : Types.ExperimentConfig -> Types.Checkpoint -> Maybe String
checkpointAddress config checkpoint =
    let
        maybeStorage =
            Utils.checkpointStorageLocation config
    in
    case ( checkpoint.state, checkpoint.uuid ) of
        ( Types.CheckpointCompleted, Just uuid ) ->
            maybeStorage
                |> Maybe.andThen
                    (\storage ->
                        (case storage of
                            Types.S3Storage bucket ->
                                [ "s3:/", bucket ]

                            Types.GcsStorage bucket ->
                                [ "gs:/", bucket ]

                            Types.SharedFSStroge hostPath Nothing ->
                                [ "file:/", hostPath ]

                            Types.SharedFSStroge hostPath (Just storagePath) ->
                                let
                                    hasAbsoluteStoragePath =
                                        String.startsWith "/" storagePath
                                in
                                if hasAbsoluteStoragePath then
                                    [ "file:/", storagePath ]

                                else
                                    [ "file:/", hostPath, storagePath ]
                        )
                            ++ [ uuid ]
                            |> String.join "/"
                            |> Just
                    )

        _ ->
            Nothing


init : Model
init =
    Closed


subscriptions : Sub Msg
subscriptions =
    Browser.Events.onKeyUp
        (D.field "key" D.string
            |> D.andThen
                (\key ->
                    if key == "Escape" then
                        D.succeed CloseModal

                    else
                        D.fail "not Escape"
                )
        )


update : Msg -> ( Model, Cmd Msg )
update msg =
    case msg of
        CloseModal ->
            ( Closed, Cmd.none )



---- View.


viewCheckpoint : Time.Zone -> Types.Experiment -> Types.Checkpoint -> H.Html Msg
viewCheckpoint zone experiment checkpoint =
    let
        resourcesTable =
            H.table [] <|
                Maybe.Extra.unwrap
                    [ H.text "N/A" ]
                    (Dict.toList
                        >> List.map
                            (\( k, v ) ->
                                H.tr [ HA.class "border-b-2 border-gray-200" ]
                                    [ H.td
                                        [ HA.class "break-words pr-4"
                                        , HA.style "max-width" "20rem"
                                        , HA.style "min-width" "10rem"
                                        ]
                                        [ H.text k ]
                                    , H.td [] [ H.text (Formatting.bytesToString v) ]
                                    ]
                            )
                    )
                    checkpoint.resources

        tupleView key value =
            H.li [ HA.class "flex flex-row content-between" ]
                [ H.span
                    [ HA.class "flex-shrink-0 mr-4 font-bold", HA.style "width" "8rem" ]
                    [ H.text key ]
                , H.span []
                    [ value ]
                ]

        validationMetricType =
            Utils.searcherValidationMetricName experiment.config

        validationMetricView =
            case ( validationMetricType, checkpoint.validationMetric ) of
                ( Just metricType, Just metricValue ) ->
                    tupleView ("Validation Metric (" ++ metricType ++ "):")
                        (H.text <| Formatting.validationFormat <| metricValue)

                _ ->
                    H.text ""

        timeView =
            Formatting.posixToString zone >> H.text

        conditionalTupleView key maybeValue =
            case maybeValue of
                Nothing ->
                    H.text ""

                Just x ->
                    tupleView key x

        body =
            [ tupleView "UUID:" (Maybe.withDefault "N/A" checkpoint.uuid |> H.text)
            , tupleView
                "State:"
                (Formatting.checkpointStateToString checkpoint.state |> H.text)
            , checkpointAddress experiment.config checkpoint
                |> Maybe.map H.text
                |> conditionalTupleView "Location:"
            , validationMetricView
            , tupleView "Total size:"
                (case checkpoint.resources of
                    Just resources ->
                        H.text <| Formatting.bytesToString <| List.sum <| Dict.values resources

                    Nothing ->
                        H.text "N/A"
                )
            , tupleView "Start time:" (timeView checkpoint.startTime)
            , tupleView "End time:"
                (Maybe.Extra.unwrap
                    (H.text "")
                    timeView
                    checkpoint.endTime
                )
            , tupleView "Resources:" resourcesTable
            ]
                |> H.ul []

        batchesPerStep =
            Utils.batchesPerStep experiment.config

        content =
            Modal.contentView
                { header =
                    H.span
                        [ HA.class "text-2xl" ]
                        [ "Checkpoint (experiment "
                            ++ String.fromInt experiment.id
                            ++ ", trial "
                            ++ String.fromInt checkpoint.trialId
                            ++ ", batch "
                            ++ String.fromInt (checkpoint.stepId * batchesPerStep)
                            ++ ")"
                            |> H.text
                        ]
                , body = body
                , footer = Nothing
                , buttons = []
                }
    in
    Modal.view
        { content = content
        , attributes = [ HA.style "width" "40rem" ]
        , closeMsg = CloseModal
        }


view : Model -> Session -> H.Html Msg
view model session =
    case model of
        OpenCheckpoint experiment checkpoint ->
            viewCheckpoint session.zone experiment checkpoint

        Closed ->
            H.text ""
