module Components.DropdownSelect exposing
    ( DropdownConfig
    , DropdownState
    , anyOfOrClear
    , clear
    , defaultInitialState
    , dropDownSelect
    , selectItems
    , selectedOrClear
    , selectionsDiffer
    , setOptions
    )

import EverySet
import Html exposing (Attribute, Html, div, i, input, span, text)
import Html.Attributes exposing (autofocus, class, placeholder, style)
import Html.Events exposing (onClick, onInput)
import Json.Decode as D


{-|

    DropdownConfig describes the behavior of the DropdownSelect component.

-}
type alias DropdownConfig a msg =
    { toMsg : DropdownState a -> msg
    , title : String
    , filterText : String
    , filtering : Bool
    , orderBySelected : Bool
    , elementToString : a -> String
    }


{-|

    DropdownState stores state for the DropdownSelect component.

-}
type alias DropdownState a =
    { open : Bool
    , filter : String
    , selectedFilters : EverySet.EverySet a
    , options : List a
    }


{-|

    isClear returns true if no elements in the DropdownSelect component are selected, otherwise
    returns false.

-}
isClear : DropdownState a -> Bool
isClear state =
    EverySet.isEmpty state.selectedFilters


{-| clear clears all selections.
-}
clear : DropdownState a -> DropdownState a
clear state =
    { state | selectedFilters = EverySet.empty }


{-|

    selected returns True if the given element is selected in the DropdownSelect.

-}
selected : DropdownState a -> a -> Bool
selected state element =
    EverySet.member element state.selectedFilters


selectedOrClear : DropdownState a -> a -> Bool
selectedOrClear state element =
    isClear state || selected state element


anyOf : DropdownState a -> List a -> Bool
anyOf state list =
    EverySet.fromList list
        |> EverySet.intersect state.selectedFilters
        |> EverySet.isEmpty
        |> not


anyOfOrClear : DropdownState a -> List a -> Bool
anyOfOrClear state list =
    isClear state || anyOf state list


{-|

    defaultInitialState returns an initial state that provides common-sense defaults for each
    parameter.

-}
defaultInitialState : List a -> DropdownState a
defaultInitialState options =
    { open = False
    , filter = ""
    , selectedFilters = EverySet.empty
    , options = options
    }


setOptions : List a -> DropdownState a -> DropdownState a
setOptions options state =
    let
        -- We have to remove any selected options that are no longer in the set of options.
        asSet =
            EverySet.fromList options

        newSelected =
            EverySet.intersect asSet state.selectedFilters
    in
    { state
        | options = options
        , selectedFilters = newSelected
    }


{-| Determine if the selections stored in the given DropdownState records are the same.
-}
selectionsDiffer : DropdownState a -> DropdownState a -> Bool
selectionsDiffer state1 state2 =
    state1.selectedFilters
        == state2.selectedFilters
        |> not


{-|

    dropdownAddOrRemoveItem updates the given state record's selectedFilters field. If
    selectedFilters contains the given element, the element will be removed from the updated state's
    selectedFilters field. Otherwise, it will be added.

-}
dropdownAddOrRemoveItem : a -> DropdownState a -> DropdownState a
dropdownAddOrRemoveItem element state =
    if EverySet.member element state.selectedFilters then
        { state | selectedFilters = EverySet.remove element state.selectedFilters }

    else
        { state | selectedFilters = EverySet.insert element state.selectedFilters }


selectItems : EverySet.EverySet a -> DropdownState a -> DropdownState a
selectItems elements state =
    let
        optionsAsSet =
            EverySet.fromList state.options

        relevantElements =
            EverySet.intersect elements optionsAsSet

        newSelected =
            EverySet.union state.selectedFilters relevantElements

        newState =
            { state | selectedFilters = newSelected }
    in
    newState


{-|

    dropdownSelectItem renders a single element in the component.

-}
dropDownSelectItem : a -> DropdownConfig a msg -> DropdownState a -> Html msg
dropDownSelectItem e config state =
    let
        label =
            config.elementToString e

        check =
            let
                maybeHidden =
                    if EverySet.member e state.selectedFilters then
                        class ""

                    else
                        class "invisible"
            in
            i [ class "text-xs pr-1 fas fa-check", maybeHidden ] []

        contents =
            [ div
                [ class "flex-grow"
                , style "min-width" "1rem"
                , style "overflow-wrap" "break-word"
                ]
                [ text label ]
            , div [ class "flex-shrink flex flex-col items-center justify-center pl-1" ]
                [ check ]
            ]

        newState =
            dropdownAddOrRemoveItem e state

        handler =
            onClickStopPropagation (config.toMsg newState)
    in
    div
        [ class "flex flex-row py-1 px-2 border-b cursor-pointer hover:bg-orange-200 bg-white w-full"
        , handler
        ]
        contents


{-|

    dropdownSelectItems renders the components elements.

-}
dropDownSelectItems : DropdownConfig a msg -> DropdownState a -> List (Html msg)
dropDownSelectItems config state =
    let
        sorter =
            \e ->
                if EverySet.member e state.selectedFilters then
                    0

                else
                    1

        options =
            if config.orderBySelected then
                List.sortBy sorter state.options

            else
                state.options
    in
    if config.filtering then
        let
            lowerFilter =
                String.toLower state.filter
        in
        List.filter (String.contains lowerFilter << String.toLower << config.elementToString) options
            |> List.map (\t -> dropDownSelectItem t config state)

    else
        options
            |> List.map (\t -> dropDownSelectItem t config state)


{-|

    dropdownSelect renders the DropdownSelect component. This is a user-facing function.

-}
dropDownSelect : DropdownConfig a msg -> DropdownState a -> Html msg
dropDownSelect config state =
    let
        selectedCount =
            EverySet.size state.selectedFilters

        countBubble =
            if selectedCount == 0 then
                text " "

            else
                span [ class "text-xs bg-gray-400 rounded-sm px-1 mx-1" ] [ text (String.fromInt selectedCount) ]

        filterHtml =
            if config.filtering then
                let
                    clickHandler =
                        onClickStopPropagation (config.toMsg state)

                    constructor =
                        \s ->
                            { state | filter = s }
                                |> config.toMsg
                in
                input
                    [ autofocus state.open
                    , placeholder "Filter"
                    , class "p-1 border"
                    , onInput constructor
                    , clickHandler
                    ]
                    []

            else
                text ""

        togglingClickHandler =
            { state | open = not state.open }
                |> config.toMsg
                |> onClickStopPropagation

        clearAllClickHandler =
            { state | selectedFilters = EverySet.empty }
                |> config.toMsg
                |> onClickStopPropagation
    in
    div [ class "relative", togglingClickHandler ]
        [ div [ class "cursor-pointer" ]
            [ span [] [ text config.title ]
            , countBubble
            , i [ class "text-xs pl-1 fas fa-chevron-down" ] []
            ]
        , if state.open then
            div [ class "relative" ]
                [ div
                    [ class "z-50 border border-gray-700 shadow absolute top-0 rounded w-64" ]
                    [ div
                        [ class "py-1 px-2 cursor-pointer text-sm border-b bg-gray-300 rounded-t border-b border-gray-400" ]
                        [ div []
                            [ span [ class "text-sm font-bold" ] [ text config.filterText ]
                            , i
                                [ class "text-xs float-right pr-1 py-1 hover:underline"
                                , clearAllClickHandler
                                ]
                                [ text "clear" ]
                            ]
                        , div [ class "rounded-b" ]
                            [ filterHtml ]
                        ]
                    , div
                        [ class "overflow-y-auto overflow-x-hidden w-full"
                        , style "max-height" "40vh"
                        ]
                        (dropDownSelectItems config state)
                    ]
                ]

          else
            text ""
        , if state.open then
            overlay config state

          else
            text ""
        ]


{-|

    onClickStopPropagation is a click handler that stops propagation of the initial click event up
    the DOM tree.

-}
onClickStopPropagation : msg -> Attribute msg
onClickStopPropagation constructor =
    Html.Events.custom "click"
        (D.succeed
            { message = constructor
            , stopPropagation = True
            , preventDefault = False
            }
        )


{-|

    overlay renders an invisible overlay that serves two functions:
        - It prevents interaction with any other UI elements.
        - It captures clicks to trigger a closing of the DropdownSelect component.

-}
overlay : DropdownConfig a msg -> DropdownState a -> Html msg
overlay config state =
    let
        newState =
            { state | open = False }

        handler =
            onClick (config.toMsg newState)
    in
    div [ class "fixed inset-0 w-full h-full z-40", handler ] []
