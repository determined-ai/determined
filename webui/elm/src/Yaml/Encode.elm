module Yaml.Encode exposing (encode)

import Dict exposing (Dict)
import Json.Decode as D
import Json.Encode as E
import Result.Extra


type YValue
    = YNull
    | YBool Bool
    | YFloat Float
    | YString String
    | YList (List YValue)
    | YDict (Dict String YValue)


jsonDecoder : D.Decoder YValue
jsonDecoder =
    let
        lazy =
            D.lazy <| \_ -> jsonDecoder
    in
    D.oneOf
        [ D.null YNull
        , D.map YBool D.bool
        , D.map YFloat D.float
        , D.map YString D.string
        , D.map YList (D.list lazy)
        , D.map YDict (D.dict lazy)
        ]


{-| Encode the given JSON value as a string.
-}
encode : E.Value -> String
encode =
    -- The default is a bit dirty, but the decoder covers all possible JSON inputs, so it'll never
    -- be used.
    D.decodeValue jsonDecoder >> Result.Extra.unwrap "" encodeInternal


{-| Encode the given internal value as a string.
-}
encodeInternal : YValue -> String
encodeInternal =
    encodeToLines
        >> List.map (\( n, l ) -> String.repeat n "  " ++ l)
        >> String.join "\n"


{-| Return the encoding of a value if it is sufficiently simple, or Nothing if
it is not. This function is used to allow simple dictionary values to be inlined
with their keys.
-}
encodeScalar : YValue -> Maybe String
encodeScalar v =
    case v of
        YNull ->
            Just "null"

        YBool b ->
            if b then
                Just "true"

            else
                Just "false"

        YFloat f ->
            Just (String.fromFloat f)

        YString s ->
            Just (E.encode 0 (E.string s))

        YList _ ->
            Nothing

        YDict d ->
            if Dict.isEmpty d then
                Just "{}"

            else
                Nothing


{-| Return a list of lines and associated indentation levels in the encoding of
the given value.
-}
encodeToLines : YValue -> List ( Int, String )
encodeToLines val =
    case val of
        YNull ->
            [ ( 0, "null" ) ]

        YBool b ->
            if b then
                [ ( 0, "true" ) ]

            else
                [ ( 0, "false" ) ]

        YFloat f ->
            [ ( 0, String.fromFloat f ) ]

        YString s ->
            [ ( 0, E.encode 0 (E.string s) ) ]

        YList l ->
            let
                transform elem =
                    case encodeToLines elem of
                        [] ->
                            []

                        ( n, x ) :: tl ->
                            ( n, "- " ++ x ) :: List.map (Tuple.mapFirst ((+) 1)) tl
            in
            if List.isEmpty l then
                [ ( 0, "[]" ) ]

            else
                List.concatMap transform l

        YDict d ->
            let
                transform elem =
                    encodeToLines elem |> List.map (Tuple.mapFirst ((+) 1))
            in
            case Dict.toList d of
                [] ->
                    [ ( 0, "{}" ) ]

                items ->
                    let
                        f ( k, v ) =
                            case encodeScalar v of
                                Just s ->
                                    [ ( 0, k ++ ": " ++ s ) ]

                                Nothing ->
                                    ( 0, k ++ ":" ) :: transform v
                    in
                    List.concatMap f items
