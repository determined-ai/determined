module CustomScalarCodecs exposing (Bytea, Checkpoint_state, Experiment_state, Float8, Id, Jsonb, Step_state, Timestamp, Timestamptz, Trial_state, Uuid, Validation_state, codecs)

import DetQL.Scalar
import Dict exposing (Dict)
import Graphql.Codec
import Iso8601
import Json.Decode as D
import Json.Encode as E
import Maybe.Extra
import Result.Extra
import Time
import Types


type alias Bytea =
    String


type alias Checkpoint_state =
    Types.CheckpointState


type alias Experiment_state =
    Types.RunState


type alias Float8 =
    Float


type alias Id =
    DetQL.Scalar.Id


type alias Jsonb =
    D.Value


type alias Step_state =
    Types.RunState


type alias Timestamp =
    Time.Posix


type alias Timestamptz =
    Time.Posix


type alias Trial_state =
    Types.RunState


type alias Uuid =
    String


type alias Validation_state =
    Types.RunState


decodeFromDict : String -> Dict String a -> D.Decoder a
decodeFromDict label dict =
    D.string
        |> D.andThen ((\s -> Dict.get s dict) >> Maybe.Extra.unwrap (D.fail ("invalid " ++ label)) D.succeed)


runStateMap : Dict String Types.RunState
runStateMap =
    Dict.fromList
        [ ( "ACTIVE", Types.Active )
        , ( "CANCELED", Types.Canceled )
        , ( "COMPLETED", Types.Completed )
        , ( "ERROR", Types.Error )
        , ( "PAUSED", Types.Paused )
        , ( "STOPPING_CANCELED", Types.StoppingCanceled )
        , ( "STOPPING_COMPLETED", Types.StoppingCompleted )
        , ( "STOPPING_ERROR", Types.StoppingError )
        ]


checkpointStateMap : Dict String Types.CheckpointState
checkpointStateMap =
    Dict.fromList
        [ ( "ACTIVE", Types.CheckpointActive )
        , ( "COMPLETED", Types.CheckpointCompleted )
        , ( "ERROR", Types.CheckpointError )
        , ( "DELETED", Types.CheckpointDeleted )
        ]


parseOneHex : Char -> Int
parseOneHex c =
    case c of
        '0' ->
            0

        '1' ->
            1

        '2' ->
            2

        '3' ->
            3

        '4' ->
            4

        '5' ->
            5

        '6' ->
            6

        '7' ->
            7

        '8' ->
            8

        '9' ->
            9

        'a' ->
            10

        'b' ->
            11

        'c' ->
            12

        'd' ->
            13

        'e' ->
            14

        'f' ->
            15

        _ ->
            0


parseHexChars : List Char -> String
parseHexChars =
    List.foldr
        (\c ( last, out ) ->
            case last of
                Nothing ->
                    ( Just c, out )

                Just d ->
                    ( Nothing, Char.fromCode (16 * parseOneHex c + parseOneHex d) :: out )
        )
        ( Nothing, [] )
        >> Tuple.second
        >> String.fromList


decodeHexString : String -> D.Decoder String
decodeHexString s =
    case String.toList s of
        '\\' :: 'x' :: rest ->
            D.succeed (parseHexChars rest)

        _ ->
            D.fail "invalid hex string"


runStateCodec : Graphql.Codec.Codec Types.RunState
runStateCodec =
    { decoder = decodeFromDict "run state" runStateMap
    , encoder = always E.null
    }


timestampCodec : Graphql.Codec.Codec Time.Posix
timestampCodec =
    { decoder =
        D.string
            |> D.andThen (Iso8601.toTime >> Result.Extra.unwrap (D.fail "invalid time") D.succeed)
    , encoder = always E.null
    }


codecs : DetQL.Scalar.Codecs Bytea Checkpoint_state Experiment_state Float8 Id Jsonb Step_state Timestamp Timestamptz Trial_state Uuid Validation_state
codecs =
    let
        default =
            DetQL.Scalar.defaultCodecs
    in
    DetQL.Scalar.defineCodecs
        { codecBytea =
            { decoder = D.string |> D.andThen decodeHexString
            , encoder = always E.null
            }
        , codecCheckpoint_state =
            { decoder = decodeFromDict "checkpoint state" checkpointStateMap
            , encoder = always E.null
            }
        , codecExperiment_state = runStateCodec
        , codecFloat8 = { decoder = D.float, encoder = E.float }
        , codecId = default.codecId
        , codecJsonb = { decoder = D.value, encoder = identity }
        , codecStep_state = runStateCodec
        , codecTimestamp = timestampCodec
        , codecTimestamptz = timestampCodec
        , codecTrial_state = runStateCodec
        , codecUuid = { decoder = D.string, encoder = E.string }
        , codecValidation_state = runStateCodec
        }
