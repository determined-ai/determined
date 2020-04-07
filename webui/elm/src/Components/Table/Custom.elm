module Components.Table.Custom exposing
    ( commandStateSorter
    , datetimeCol
    , emptyCell
    , htmlUnsortableColumn
    , intCol
    , intColD
    , labelsCol
    , limitedWidthStringColWithIcon
    , makeCustomCol
    , maybeNumericalSorter
    , percentCol
    , runStateCol
    , stringCol
    , stringColD
    , tableCustomizations
    )

import Components.LabelWidget as LW
import Components.Table as Table
import Constants
import Formatting exposing (millisToString, toPct)
import Html as H exposing (span, text)
import Html.Attributes as HA
import Maybe.Extra
import Page.Common exposing (runStateToSpan)
import Time
import Types



---- Helpers


emptyCell : H.Html msg
emptyCell =
    H.text ""



---- Custom columns.


makeCustomCol :
    List (H.Attribute msg)
    -> (comparable -> String)
    -> ((a -> comparable) -> Table.Sorter a)
    -> String
    -> String
    -> (a -> comparable)
    -> Table.Column a msg
makeCustomCol additionalAtts toStr sorter name id getter =
    let
        viewData res =
            { attributes = HA.class "p-2" :: additionalAtts
            , children = [ res |> getter |> toStr |> text ]
            }
    in
    Table.veryCustomColumn
        { name = name
        , id = id
        , viewData = viewData
        , sorter = sorter getter
        }


stringCol : String -> String -> (a -> String) -> Table.Column a msg
stringCol =
    makeCustomCol [] identity Table.increasingOrDecreasingByString


limitedWidthAtts : String -> List (H.Attribute msg)
limitedWidthAtts maxWidth =
    [ HA.style "overflow-wrap" "break-word"
    , HA.style "max-width" maxWidth
    ]


limitedWidthStringColWithIcon : String -> String -> String -> (a -> String) -> (a -> Bool) -> Table.Column a msg
limitedWidthStringColWithIcon maxWidth name id toString toBool =
    let
        viewData d =
            { attributes = limitedWidthAtts maxWidth
            , children =
                [ H.div [ HA.class "" ]
                    [ if toBool d then
                        H.i
                            [ HA.class "text-gray-600 text-xs fas fa-archive"
                            , HA.title "Archived"
                            ]
                            []

                      else
                        H.text ""
                    , H.span [ HA.class "px-1" ] [ toString d |> H.text ]
                    ]
                ]
            }
    in
    Table.veryCustomColumn
        { id = id
        , name = name
        , viewData = viewData
        , sorter = Table.increasingOrDecreasingByString toString
        }


stringColD : String -> String -> (a -> String) -> Table.Column a msg
stringColD =
    makeCustomCol [] identity Table.increasingOrDecreasingByString


intCol : String -> String -> (a -> Int) -> Table.Column a msg
intCol =
    makeCustomCol [] String.fromInt Table.increasingOrDecreasingBy


intColD : String -> String -> (a -> Int) -> Table.Column a msg
intColD =
    makeCustomCol [] String.fromInt Table.decreasingOrIncreasingBy


percentCol : String -> String -> (a -> Float) -> Table.Column a msg
percentCol =
    makeCustomCol [] toPct Table.decreasingOrIncreasingBy


labelsCol : LW.Config msg -> Table.Column ( Types.ExperimentResult, LW.State ) msg
labelsCol config =
    let
        viewData res =
            { attributes = [ HA.class "p-2" ]
            , children = viewExp res
            }

        viewExp res =
            case res of
                ( Ok exp, state ) ->
                    [ LW.view config state exp.labels ]

                ( Err _, _ ) ->
                    []
    in
    Table.veryCustomColumn
        { name = "Labels"
        , id = "labels"
        , viewData = viewData
        , sorter = Table.unsortable
        }


datetimeCol : Time.Zone -> String -> String -> (a -> Int) -> Table.Column a msg
datetimeCol zone =
    makeCustomCol [] (millisToString zone) Table.decreasingOrIncreasingBy


viewRunState : Maybe Types.RunState -> Table.HtmlDetails msg
viewRunState state =
    let
        html =
            Maybe.Extra.unwrap (H.text "") runStateToSpan state
    in
    { children = [ html ], attributes = [ HA.class "p-2" ] }


runStateCol : (data -> Maybe Types.RunState) -> Table.Column data msg
runStateCol stateGetter =
    Table.veryCustomColumn
        { name = "State"
        , id = "state"
        , viewData = stateGetter >> viewRunState
        , sorter = Table.increasingOrDecreasingBy (Maybe.Extra.unwrap 8 runStateSorter << stateGetter)
        }


runStateSorter : Types.RunState -> Int
runStateSorter runState =
    case runState of
        Types.Active ->
            0

        Types.Paused ->
            1

        Types.StoppingError ->
            2

        Types.Error ->
            3

        Types.StoppingCompleted ->
            4

        Types.Completed ->
            5

        Types.StoppingCanceled ->
            6

        Types.Canceled ->
            7


{-| commandStateSorter defines the ordering of commands based on state. This is used by command,
notebook, and TensorBoard tables.
-}
commandStateSorter : Types.CommandState -> Int
commandStateSorter state =
    case state of
        Types.CmdPending ->
            0

        Types.CmdAssigned ->
            1

        Types.CmdPulling ->
            2

        Types.CmdStarting ->
            3

        Types.CmdRunning ->
            4

        Types.CmdTerminating ->
            5

        Types.CmdTerminated ->
            6


maybeNumericalSorter : (data -> Maybe Float) -> Bool -> Table.Sorter data
maybeNumericalSorter mapper smallerIsBetter =
    if smallerIsBetter then
        Table.increasingOrDecreasingBy (mapper >> Maybe.withDefault Constants.infinity)

    else
        Table.decreasingOrIncreasingBy (mapper >> Maybe.withDefault -Constants.infinity)


htmlUnsortableColumn : String -> String -> (data -> Table.HtmlDetails msg) -> Table.Column data msg
htmlUnsortableColumn name id html =
    Table.veryCustomColumn
        { name = name
        , id = id
        , viewData = html
        , sorter = Table.unsortable
        }



---- Custom formatting.


darkGrey : String -> H.Html msg
darkGrey symbol =
    H.span [ HA.style "color" "#555" ] [ H.text (" " ++ symbol) ]


lightGrey : String -> H.Html msg
lightGrey symbol =
    H.span [ HA.style "color" "#ccc" ] [ H.text (" " ++ symbol) ]


simpleTheadHelp : ( String, Table.Status, H.Attribute msg ) -> H.Html msg
simpleTheadHelp ( name, status, onClick_ ) =
    let
        content =
            case status of
                Table.Unsortable ->
                    [ H.text name ]

                Table.Sortable selected ->
                    [ H.text name
                    , if selected then
                        darkGrey "↓"

                      else
                        lightGrey "↓"
                    ]

                Table.Reversible Nothing ->
                    [ H.text name
                    , lightGrey "↕"
                    ]

                Table.Reversible (Just isReversed) ->
                    [ H.text name
                    , darkGrey
                        (if isReversed then
                            "↑"

                         else
                            "↓"
                        )
                    ]
    in
    H.th [ HA.class "px-2", HA.style "text-align" "left", onClick_ ] content


generateTableHeader : List ( String, Table.Status, H.Attribute msg ) -> Table.HtmlDetails msg
generateTableHeader ls =
    { attributes = [ HA.style "background-color" "#e2e8f0" ]
    , children = List.map simpleTheadHelp ls
    }


tableCustomizations : Table.Customizations data msg
tableCustomizations =
    { tableAttrs = [ HA.class "w-full", HA.style "min-width" "max-content" ]
    , caption = Nothing
    , thead = generateTableHeader
    , tfoot = Nothing
    , tbodyAttrs = []
    , rowAttrs = always [ HA.class "hover:bg-orange-100" ]
    }
