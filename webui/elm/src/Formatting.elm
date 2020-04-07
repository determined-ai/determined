module Formatting exposing
    ( boolToString
    , bytesToString
    , checkpointStateToString
    , commandStateToString
    , compactFloat
    , durationToString
    , maybeAddNewLine
    , millisToString
    , posixToString
    , runStateToString
    , toPct
    , truncateString
    , validationFormat
    )

{-| Various small formatting-related functions.
-}

import DateFormat
import Duration
import Filesize
import FormatNumber
import FormatNumber.Locales exposing (usLocale)
import List.Extra
import Round
import String.Extra
import Time
import Types


boolToString : Bool -> String
boolToString b =
    if b then
        "true"

    else
        "false"


posixToString : Time.Zone -> Time.Posix -> String
posixToString =
    DateFormat.format
        [ DateFormat.yearNumber
        , DateFormat.text "/"
        , DateFormat.monthFixed
        , DateFormat.text "/"
        , DateFormat.dayOfMonthFixed
        , DateFormat.text ", "
        , DateFormat.hourNumber
        , DateFormat.text ":"
        , DateFormat.minuteFixed
        , DateFormat.text ":"
        , DateFormat.secondFixed
        , DateFormat.text " "
        , DateFormat.amPmUppercase
        ]


toPct : Float -> String
toPct n =
    -- cheat and use invalid percentiles as a sentinels.
    if (n < 0) || (n > 100) then
        ""

    else
        FormatNumber.format { usLocale | decimals = 1 } n ++ "%"


millisToString : Time.Zone -> Int -> String
millisToString zone millis =
    if millis < 0 then
        ""

    else
        Time.millisToPosix millis |> posixToString zone


commandStateToString : Types.CommandState -> String
commandStateToString state =
    case state of
        Types.CmdPending ->
            "Pending"

        Types.CmdAssigned ->
            "Assigned"

        Types.CmdPulling ->
            "Pulling"

        Types.CmdStarting ->
            "Starting"

        Types.CmdRunning ->
            "Running"

        Types.CmdTerminating ->
            "Terminating"

        Types.CmdTerminated ->
            "Terminated"


runStateToString : Types.RunState -> String
runStateToString state =
    case state of
        Types.Active ->
            "Active"

        Types.Canceled ->
            "Canceled"

        Types.Completed ->
            "Completed"

        Types.Error ->
            "Errored"

        Types.Paused ->
            "Paused"

        Types.StoppingCanceled ->
            "Cancelling"

        Types.StoppingCompleted ->
            "Completing"

        Types.StoppingError ->
            "Erroring"


checkpointStateToString : Types.CheckpointState -> String
checkpointStateToString state =
    case state of
        Types.CheckpointActive ->
            "Active"

        Types.CheckpointError ->
            "Errored"

        Types.CheckpointCompleted ->
            "Completed"

        Types.CheckpointDeleted ->
            "Deleted"


{-| TODO: This method has a bug: if given, e.g., "1000", it returns "1".
-}
trimNumber : String -> String
trimNumber =
    String.toList
        >> List.Extra.dropWhileRight ((==) '0')
        >> List.Extra.dropWhileRight ((==) '.')
        >> String.fromList


truncateString : Int -> String -> String
truncateString limit str =
    String.Extra.break limit str
        |> List.head
        |> Maybe.withDefault ""


bytesToString : Int -> String
bytesToString bytes =
    let
        ( num, unit ) =
            Filesize.formatWithSplit Filesize.defaultSettings bytes
    in
    num ++ unit


validationFormat : Float -> String
validationFormat =
    compactFloat 4 6


{-| For displaying numbers compactly.

  - maxNonSciDisplay: The maximum magnitude of number before switching to scientific notation.
      - For numbers >= 1, this is the number of places before the decimal point.
      - For numbers < 1, this is the number of places after the decimal point, up to and including
        the first non-zero digit.
  - maxDecimalPlaces: The maximum number of decimal places to display, scientific or otherwise.
      - For small numbers, this should be greater than maxNonSciDisplay, otherwise it will display "0".

-}
compactFloat : Int -> Int -> Float -> String
compactFloat maxNonSciDisplay maxDecimalPlaces x =
    let
        mag =
            abs x

        sign =
            if x < 0 then
                "−"

            else
                ""

        log0 =
            if x == 0 then
                0

            else
                floor (logBase 10 mag)

        significand0 =
            mag
                / toFloat (10 ^ log0)
                |> Round.round maxDecimalPlaces
                |> trimNumber

        -- Do another division if rounding the significand bumped it up to 10, so that we get,
        -- e.g., "1e5" and not "10e4".
        ( log, significand ) =
            let
                s =
                    String.toFloat significand0 |> Maybe.withDefault 0
            in
            if s < 10 then
                ( log0, significand0 )

            else
                ( log0 + 1, String.fromFloat (s / 10) |> trimNumber )
    in
    if log >= -maxNonSciDisplay && log < maxNonSciDisplay then
        -- Use a basic format for numbers of middling magnitude.
        sign ++ (Round.round maxDecimalPlaces mag |> trimNumber)

    else
        -- Use scientific notation for numbers of large or small magnitude.
        let
            significandNum =
                String.toFloat significand |> Maybe.withDefault 0

            expString =
                "e" ++ (String.fromInt log |> String.replace "-" "−")
        in
        sign ++ (Round.round maxDecimalPlaces significandNum |> trimNumber) ++ expString


durationToString : Duration.Duration -> String
durationToString duration =
    let
        seconds aDuration =
            let
                numerical =
                    Duration.inSeconds aDuration
                        |> round
                        |> modBy 60
            in
            if numerical == 0 then
                "< 1s"

            else
                String.fromInt numerical ++ "s"

        minutes aDuration =
            let
                numerical =
                    Duration.inMinutes aDuration
                        |> floor
                        |> modBy 60
            in
            if numerical == 0 then
                ""

            else
                String.fromInt numerical ++ "m"

        hours aDuration =
            let
                numerical =
                    Duration.inHours aDuration
                        |> floor
                        |> modBy 24
            in
            if numerical == 0 then
                ""

            else
                String.fromInt numerical ++ "h"

        days aDuration =
            let
                numerical =
                    Duration.inDays aDuration
                        |> floor
            in
            if numerical == 0 then
                ""

            else
                String.fromInt numerical ++ "d"

        timeSegments =
            if Duration.inDays duration > 1 then
                [ days duration
                , hours duration
                , minutes duration
                ]

            else
                [ hours duration
                , minutes duration
                , seconds duration
                ]
    in
    List.filter ((/=) "")
        timeSegments
        |> List.intersperse " "
        |> String.concat


{-| Add a '\\n' to the end of the string if one is not already present.
-}
maybeAddNewLine : String -> String
maybeAddNewLine s =
    if String.isEmpty s then
        ""

    else if not (String.endsWith "\n" s) then
        s ++ "\n"

    else
        s
