module RecordState exposing
    ( findRecordState
    , getRecords
    , updateRecordState
    , updateRecordStates
    )

{-| This module provides helpers to manage metadata around each inidividual record in a list
of records. Namely, it can be used to track the state of stateful components for each individual
record.
-}

import Dict
import List.Extra
import Maybe.Extra


{-| RecordState defines a record, usually in a list of records, coming from an API call that
we are interested in keeping some related state for.

    recordWithMetadata {
        record: {id: x, ...} -- coming from outside
        a: -- related state
        b: -- related state

-}
type alias RecordState a b =
    { a | record : Record b }


type alias Record a =
    { a | id : String }


isSameRecordState : String -> RecordState a b -> Bool
isSameRecordState recordID rState =
    rState.record
        |> .id
        |> (==) recordID


findRecordState :
    Maybe (List (RecordState a b))
    -> String
    -> Maybe (RecordState a b)
findRecordState rStatesMaybe recordID =
    rStatesMaybe
        |> Maybe.andThen
            (List.Extra.find (isSameRecordState recordID))


{-| Updates a single recordState from a list of them.
-}
updateRecordState :
    RecordState a b
    -> Maybe (List (RecordState a b))
    -> Maybe (List (RecordState a b))
updateRecordState recordState =
    Maybe.map
        (List.Extra.updateIf
            (isSameRecordState recordState.record.id)
            (always recordState)
        )


updateRecordStates :
    Maybe (List (RecordState a b))
    -> List (Record b)
    -> Maybe (RecordState a b -> Record b -> RecordState a b)
    -> (Record b -> RecordState a b)
    -> Maybe (List (RecordState a b))
updateRecordStates recordStatesMaybe records updateFnMaybe initFn =
    let
        existingRecordStates =
            Maybe.Extra.unwrap
                Dict.empty
                (List.map (\recState -> ( recState.record.id, recState )) >> Dict.fromList)
                recordStatesMaybe

        mapper : Record b -> RecordState a b
        mapper record =
            case Dict.get record.id existingRecordStates of
                Just recordState ->
                    case updateFnMaybe of
                        Just updateFn ->
                            updateFn recordState record

                        Nothing ->
                            { recordState | record = record }

                Nothing ->
                    initFn record
    in
    if List.length records == 0 then
        Nothing

    else
        Just (List.map mapper records)


getRecords : List (RecordState a b) -> List (Record b)
getRecords recordStates =
    List.map .record recordStates
