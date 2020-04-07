module View.SlotChart exposing
    ( allocationPercent
    , largeView
    , smallView
    )

import Dict exposing (Dict)
import Html as H
import Html.Attributes as HA
import Types


type alias Config =
    { showTotal : Bool
    , showLegend : Bool
    }


largeConfig : Config
largeConfig =
    { showTotal = True
    , showLegend = True
    }


smallConfig : Config
smallConfig =
    { showTotal = False
    , showLegend = False
    }


{-| Create a histogram from a list of values of any type, given a function mapping from that type
into a comparable type to use as keys.
-}
frequency : (a -> comparable) -> List a -> Dict comparable Int
frequency toString =
    List.foldl
        (\k -> Dict.update (toString k) (Maybe.withDefault 0 >> (+) 1 >> Just))
        Dict.empty


{-| Colors drawn from the Glasbey categorical color palette:
<https://strathprints.strath.ac.uk/30312/1/colorpaper_2006.pdf>.
-}
slotStateColor : Types.SlotState -> String
slotStateColor state =
    case state of
        Types.Assigned ->
            "#2fa1da"

        Types.Pulling ->
            "#fb4f2f"

        Types.Starting ->
            "#e4ae38"

        Types.Running ->
            "#6d904f"

        Types.Terminating ->
            "#9367bc"

        Types.Terminated ->
            "#d62628"

        Types.Free ->
            "#8a8a8a"


slotStateName : Types.SlotState -> String
slotStateName state =
    case state of
        Types.Assigned ->
            "Assigned"

        Types.Pulling ->
            "Pulling"

        Types.Starting ->
            "Starting"

        Types.Running ->
            "Running"

        Types.Terminating ->
            "Terminating"

        Types.Terminated ->
            "Terminated"

        Types.Free ->
            "Free"


allocationChart : Config -> List Types.Slot -> H.Html msg
allocationChart config slots =
    let
        { showTotal, showLegend } =
            config

        numSlots =
            List.length slots

        -- The slot states in the order they appear in the chart, left to right.
        statesInOrder =
            [ Types.Assigned
            , Types.Pulling
            , Types.Starting
            , Types.Running
            , Types.Terminating
            , Types.Terminated
            , Types.Free
            ]

        -- A map from each state to the number of slots in that state.
        counts =
            frequency slotStateName (List.map .state slots)

        -- For each state in order, a tuple containing the state and the number of slots in that
        -- state (if there are any).
        countsInOrder =
            statesInOrder
                |> List.filterMap
                    (\state ->
                        Dict.get (slotStateName state) counts |> Maybe.map (\n -> ( state, n ))
                    )

        title =
            if showTotal then
                H.div [ HA.class "text-center font-bold pb-2" ]
                    [ H.text ("Total slots: " ++ String.fromInt numSlots) ]

            else
                H.text ""

        bar =
            countsInOrder
                |> List.map
                    (\( state, count ) ->
                        H.div
                            [ HA.style "width" (String.fromFloat (100 * toFloat count / toFloat numSlots) ++ "%")
                            , HA.style "background" (slotStateColor state)
                            ]
                            []
                    )
                |> H.div [ HA.class "flex flex-row flex-grow", HA.style "min-height" ".5rem" ]

        legend =
            if showLegend then
                let
                    spacer =
                        H.span [ HA.class "mx-4" ] []
                in
                countsInOrder
                    |> List.map
                        (\( state, count ) ->
                            H.span [ HA.class "inline-flex items-center" ]
                                [ H.span
                                    [ HA.class "p-2"
                                    , HA.style "background" (slotStateColor state)
                                    ]
                                    []
                                , H.span [ HA.class "ml-2" ]
                                    [ H.text (slotStateName state ++ ": " ++ String.fromInt count) ]
                                ]
                        )
                    |> List.intersperse spacer
                    |> H.div [ HA.class "text-center pt-2" ]

            else
                H.text ""
    in
    -- The title and legend have a fixed size at the top and bottom of the container, while the bar
    -- grows vertically to fill the remaining space.
    H.div [ HA.class "w-full h-full flex flex-col" ]
        [ title
        , bar
        , legend
        ]


smallView : List Types.Slot -> H.Html msg
smallView =
    allocationChart smallConfig


allocationPercent : List Types.Slot -> String
allocationPercent slots =
    let
        totalSlots =
            List.length slots

        busySlots =
            List.filter (\slot -> slot.state /= Types.Free) slots
                |> List.length

        precision =
            10 ^ 1

        percentNum =
            toFloat busySlots
                / toFloat totalSlots
                |> (*) 100
                |> (*) precision
                |> round
    in
    (if List.length slots == 0 then
        "0"

     else if modBy precision percentNum == 0 then
        percentNum
            // precision
            |> String.fromInt

     else
        toFloat percentNum
            / precision
            |> String.fromFloat
    )
        ++ "%"


largeView : List Types.Slot -> H.Html msg
largeView =
    allocationChart largeConfig
