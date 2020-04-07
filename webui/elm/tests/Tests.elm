module Tests exposing (formatting, slotChart)

import Expect
import Formatting
import Maybe.Extra
import Test exposing (..)
import Types
import View.SlotChart


formatting : Test
formatting =
    describe "compactFloat" <|
        let
            compactFloatTest x maxNonSciDisplay maxDecimalPlaces expectedString =
                Formatting.compactFloat maxNonSciDisplay maxDecimalPlaces x
                    |> Expect.equal expectedString
        in
        [ test "shows decimal places to maxDecimalPlaces" <|
            \_ ->
                compactFloatTest 0.111222 4 4 "0.1112"
        , test "rounds at the end of maxDecimalPlaces" <|
            \_ ->
                compactFloatTest 0.123456 4 4 "0.1235"
        , test "maintains basic format for small numbers up to maxNonSciDisplay" <|
            \_ ->
                compactFloatTest 0.0001234 4 4 "0.0001"
        , test "maintains basic format for small negative numbers up to maxNonSciDisplay" <|
            \_ ->
                compactFloatTest -0.0001234 4 4 "−0.0001"
        , test "switches to scientific for small numbers with >= zeroes than maxNonSciDisplay" <|
            \_ ->
                compactFloatTest 0.0001234 3 4 "1.234e−4"
        , test "switches to scientific for small negative numbers with >= zeroes than maxNonSciDisplay" <|
            \_ ->
                compactFloatTest -0.0001234 3 4 "−1.234e−4"
        , test "maintains basic format for large numbers up to maxNonSciDisplay" <|
            \_ ->
                compactFloatTest 1234 4 4 "1234"
        , test "switches to scientific for large numbers with more figures than maxNonSciDisplay" <|
            \_ ->
                compactFloatTest 1234 3 4 "1.234e3"
        , test "maintains basic format for large numbers up to maxNonSciDisplay and respects maxDecimalPlaces" <|
            \_ ->
                compactFloatTest 1234.5678 4 2 "1234.57"
        , test "maintains basic format for large negative numbers up to maxNonSciDisplay and respects maxDecimalPlaces" <|
            \_ ->
                compactFloatTest -1234.5678 4 2 "−1234.57"
        , test "switches to scientific for large numbers with more figures than maxNonSciDisplay and respects maxDecimalPlaces" <|
            \_ ->
                compactFloatTest 1234.5678 3 2 "1.23e3"
        , test "handles 0 well" <|
            \_ ->
                compactFloatTest 0 3 2 "0"
        ]


slotChart : Test
slotChart =
    describe "slotChart" <|
        let
            allocationPercentTest expectedString slots =
                View.SlotChart.allocationPercent slots
                    |> Expect.equal expectedString

            freeSlots =
                List.range 0 2
                    |> List.map
                        (\id ->
                            { id = String.fromInt id
                            , slotType = Types.GPU
                            , device = "fancyGPU"
                            , state = Types.Free
                            }
                        )
        in
        [ test "shows 0% for no slots" <|
            \_ ->
                allocationPercentTest "0%" []
        , test "shows 0% for all free slots" <|
            \_ ->
                allocationPercentTest "0%" freeSlots
        , test "shows 100% for all occupied slots" <|
            \_ ->
                freeSlots
                    |> List.map
                        (\slot ->
                            { slot | state = Types.Running }
                        )
                    |> allocationPercentTest "100%"
        , test "shows 33.3% for 1/3 occupied slots" <|
            \_ ->
                let
                    slots =
                        [ { id = "x"
                          , slotType = Types.GPU
                          , device = "fancyGPU"
                          , state = Types.Running
                          }
                        , { id = "x"
                          , slotType = Types.GPU
                          , device = "fancyGPU"
                          , state = Types.Free
                          }
                        , { id = "x"
                          , slotType = Types.GPU
                          , device = "fancyGPU"
                          , state = Types.Free
                          }
                        ]
                in
                allocationPercentTest "33.3%" slots
        ]
