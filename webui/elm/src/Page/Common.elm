module Page.Common exposing
    ( ButtonAction(..)
    , ButtonConfig
    , ButtonStyle(..)
    , bigMessage
    , breadcrumb
    , buttonCreator
    , centeredLoadingWidget
    , commandStateToSpan
    , headerClasses
    , horizontalList
    , isCommandOpenable
    , killButton
    , killButtonConfig
    , maybeLoadingOverlay
    , onClickStopPropagation
    , openButton
    , openButtonConfig
    , openLogsButton
    , openWaitPage
    , pageHeader
    , runStateToSpan
    , section
    , selectFromValues
    , spinner
    , unruledSection
    , verticalCollapseButton
    )

import API
import Formatting exposing (commandStateToString, runStateToString)
import Html as H
import Html.Attributes as HA
import Html.Events as HE
import Json.Decode as D
import Maybe.Extra
import Ports
import Types
import Url.Builder as UB
import Utils


bigMessage : String -> H.Html msg
bigMessage text =
    H.div [ HA.class "w-full text-center text-gray-500 text-3xl p-4" ] [ H.text text ]


baseButtonClasses : String
baseButtonClasses =
    String.join " "
        [ "font-bold" -- Style text.
        , "py-1 px-2" -- Set up spacing.
        , "disabled:opacity-25 disabled:cursor-not-allowed" -- Make disabled buttons distinct.
        , "focus:outline-none focus:shadow-outline" -- Set up nicely visible focus outlines.
        , "smooth-opacity rounded-sm" -- Set up opacity animation/corners.
        ]


{-| Create a class suitable to use for a button. The argument is a string containing extra classes
to apply (typically a background color).
-}
buttonClass : String -> H.Attribute msg
buttonClass c =
    HA.class <| baseButtonClasses ++ " " ++ c


headerClasses : String
headerClasses =
    "whitespace-no-wrap text-gray-700 font-bold pb-2 "


pageHeader : String -> H.Html msg
pageHeader text =
    H.div [ headerClasses ++ "text-2xl" |> HA.class ] [ H.text text ]


sectionHeader : String -> H.Html msg
sectionHeader text =
    H.div [ headerClasses ++ "text-xl" |> HA.class ] [ H.text text ]


section : String -> List (H.Html msg) -> H.Html msg
section title children =
    H.div [ HA.class "py-2 border-t" ] <| sectionHeader title :: children


unruledSection : String -> List (H.Html msg) -> H.Html msg
unruledSection title children =
    H.div [ HA.class "py-2" ] <| sectionHeader title :: children


runStateToStyle : Types.RunState -> List (H.Attribute msg)
runStateToStyle state =
    case state of
        Types.Active ->
            [ HA.class "text-blue-500" ]

        Types.Canceled ->
            [ HA.class "text-yellow-500" ]

        Types.Completed ->
            [ HA.class "text-green-500" ]

        Types.Error ->
            [ HA.class "text-red-500" ]

        Types.Paused ->
            [ HA.class "text-yellow-500" ]

        Types.StoppingCanceled ->
            [ HA.class "text-yellow-500" ]

        Types.StoppingCompleted ->
            [ HA.class "text-green-500" ]

        Types.StoppingError ->
            [ HA.class "text-red-500" ]


runStateToSpan : Types.RunState -> H.Html msg
runStateToSpan state =
    H.span (runStateToStyle state)
        [ runStateToString state |> H.text
        ]


commandStateToStyle : Types.CommandState -> List (H.Attribute msg)
commandStateToStyle state =
    case state of
        Types.CmdPending ->
            [ HA.class "text-yellow-500" ]

        Types.CmdAssigned ->
            [ HA.class "text-yellow-500" ]

        Types.CmdPulling ->
            [ HA.class "text-yellow-500" ]

        Types.CmdStarting ->
            [ HA.class "text-yellow-500" ]

        Types.CmdRunning ->
            [ HA.class "text-green-500" ]

        Types.CmdTerminating ->
            [ HA.class "text-yellow-500" ]

        Types.CmdTerminated ->
            [ HA.class "text-green-500" ]


commandStateToSpan : Types.CommandState -> H.Html msg
commandStateToSpan state =
    H.span (commandStateToStyle state)
        [ commandStateToString state |> H.text ]


{-| String in the Full and IconOnly button styles represents the icon class. If the style is
IconOnly then the text will get rendered as the title.
-}
type ButtonStyle
    = Full String
    | IconOnly String
    | TextOnly


type ButtonAction msg
    = OpenUrl Bool String -- penInNewTab and URL
    | SendMsg msg


type alias ButtonConfig msg =
    { action : ButtonAction msg

    -- Use "transparent" for transparent background.
    , bgColor : String
    , fgColor : String
    , isActive : Bool
    , isPending : Bool
    , style : ButtonStyle
    , text : String
    }


buttonCreator : ButtonConfig msg -> H.Html msg
buttonCreator btnConfig =
    -- FIXME(hamidzr): White color isn't directly supported with how the color is passed in now.
    -- Also it might be a good idea to refactor this to have config.text also be a parameter of the
    -- ButtonStyle.
    let
        icon iconClass =
            H.i
                [ HA.class
                    (iconClass
                        ++ " my-auto"
                        ++ (case btnConfig.style of
                                Full _ ->
                                    " mr-1"

                                _ ->
                                    ""
                           )
                    )
                ]
                []

        text =
            H.span [] [ H.text btnConfig.text ]

        pendingSpinner =
            if btnConfig.isPending then
                spinner

            else
                H.text ""

        ( content, title ) =
            case btnConfig.style of
                Full iconClass ->
                    ( [ icon iconClass
                      , text
                      , pendingSpinner
                      ]
                    , HA.title ""
                    )

                IconOnly iconClass ->
                    ( [ icon iconClass
                      , pendingSpinner
                      ]
                    , HA.title btnConfig.text
                    )

                TextOnly ->
                    ( [ text
                      , pendingSpinner
                      ]
                    , HA.title ""
                    )

        class =
            " hover:shadow focus:shadow"
                ++ (" bg-" ++ btnConfig.bgColor ++ "-500")
                -- A hacky way to cover bg-black and bg-white tailwind classes.
                ++ (" bg-" ++ btnConfig.bgColor)
                ++ (" text-" ++ btnConfig.fgColor ++ "-400")
                -- A hacky way to cover text-black and text-white tailwind classes.
                ++ (" text-" ++ btnConfig.fgColor)
                ++ (if btnConfig.isActive then
                        " hover:bg-"
                            ++ btnConfig.bgColor
                            ++ "-700"

                    else
                        -- These are the same as the "disabled:"-prefixed classes in `buttonClass`,
                        -- but the disabled state can't apply to <a> elements, so explicitly
                        -- include them here when necessary.
                        " opacity-25 cursor-not-allowed"
                   )

        contentWrapper =
            [ H.div
                [ HA.class "inline-flex jusitfy-center" ]
                content
            ]

        buttonCssStyles =
            [ HA.style "min-width" "2em"
            , HA.style "min-height" "2em"
            ]
    in
    case btnConfig.action of
        OpenUrl openInNewTab url ->
            -- Using H.a to render a newly inactive button causes the href
            -- attribute to remain on the anchor tag, which allows it to remain
            -- clickable. This might be due to how Elm does a Virtual DOM diff.
            -- The desirable behavior of removing the href attribute completely
            -- is not possible. Alternative approach was to render the disabled
            -- anchor button as an actual button.
            if btnConfig.isActive then
                H.a
                    ([ buttonClass (class ++ " inline-block")
                     , title
                     , HA.target <| Utils.ifThenElse openInNewTab "_blank" "_self"
                     , HA.href url
                     ]
                        ++ buttonCssStyles
                    )
                    contentWrapper

            else
                H.button
                    ([ buttonClass class
                     , title
                     , HA.disabled True
                     ]
                        ++ buttonCssStyles
                    )
                    contentWrapper

        SendMsg msg ->
            H.button
                ([ buttonClass class
                 , title
                 , onClickStopPropagation msg
                 , HA.disabled (not btnConfig.isActive)
                 ]
                    ++ buttonCssStyles
                )
                contentWrapper


{-| TODO(hamidzr): Refactor horizontalList to take an optional separator.
-}
horizontalList : List (H.Html msg) -> H.Html msg
horizontalList items =
    List.map
        (\it -> H.li [] [ it ])
        items
        |> H.ul [ HA.class "horizontal-list" ]


openButtonConfig : ButtonAction msg -> Bool -> ButtonConfig msg
openButtonConfig action isActive =
    { action = action
    , bgColor = "orange"
    , fgColor = "white"
    , isActive = isActive
    , isPending = False
    , style = Full "fas fa-external-link-alt"
    , text = "Open"
    }


openButton : ButtonAction msg -> Bool -> H.Html msg
openButton action isActive =
    openButtonConfig action isActive
        |> buttonCreator


killButtonConfig : msg -> Bool -> ButtonConfig msg
killButtonConfig msg isActive =
    { action = SendMsg msg
    , bgColor = "red"
    , fgColor = "white"
    , isActive = isActive
    , isPending = False
    , style = Full "fa fa-ban mr-1"
    , text = "Kill"
    }


killButton : msg -> Bool -> H.Html msg
killButton msg isActive =
    killButtonConfig msg isActive
        |> buttonCreator


verticalCollapseButton : Bool -> msg -> H.Html msg
verticalCollapseButton show msg =
    let
        ( btnText, btnIcon ) =
            if show then
                ( "Hide", "fas fa-chevron-up" )

            else
                ( "Show", "fas fa-chevron-down" )
    in
    buttonCreator
        { action = SendMsg msg
        , bgColor = "transparent"
        , fgColor = "grey"
        , isActive = True
        , isPending = False
        , style = IconOnly btnIcon
        , text = btnText
        }


isCommandOpenable : Types.CommandState -> Bool
isCommandOpenable state =
    case state of
        Types.CmdTerminated ->
            False

        Types.CmdTerminating ->
            False

        _ ->
            True


{-| loadingWidget renders a loading spinner. Positioning is up to the caller.
-}
loadingWidget : H.Html msg
loadingWidget =
    H.div [ HA.class "lds-ellipsis" ]
        [ H.div [] []
        , H.div [] []
        , H.div [] []
        ]


{-| centeredLoadingWidget renders a loading spinner that is centered within its parent.
-}
centeredLoadingWidget : H.Html msg
centeredLoadingWidget =
    H.div [ HA.class "absolute inset-0 flex justify-center items-center" ] [ loadingWidget ]


{-| maybeLoadingOverlay renders a loading spinner and grays out the background.
-}
maybeLoadingOverlay : Bool -> H.Html msg
maybeLoadingOverlay isActive =
    if isActive then
        H.div [ HA.class "fixed inset-0 flex items-center align-center justify-center" ]
            [ H.div [ HA.class " bg-gray-900 opacity-50 absolute inset-0" ] []
            , H.div [ HA.class "" ]
                [ loadingWidget ]
            ]

    else
        H.text ""


{-| spinner renders a loading spinner.
-}
spinner : H.Html msg
spinner =
    H.span [ HA.class "loading-spinner ml-1 my-auto" ] []


{-| onClickStopPropagation is a click handler that stops propagation of the initial click event up
the DOM tree.
-}
onClickStopPropagation : msg -> H.Attribute msg
onClickStopPropagation msg =
    HE.stopPropagationOn "click" (D.succeed ( msg, True ))


{-| Create a select element from a union type.
Idea taken from <https://gurdiga.github.io/blog/2017/07/09/select-from-union-type-in-elm/>.
-}
selectFromValues : List ( a, String ) -> a -> (a -> msg) -> H.Html msg
selectFromValues valuesWithLabels defaultValue toMsg =
    let
        optionForTuple ( value, label ) =
            H.option [ HA.selected (defaultValue == value) ] [ H.text label ]

        valueFromLabel l =
            List.filter (\( _, label ) -> label == l) valuesWithLabels
                |> List.head
                |> Maybe.Extra.unwrap defaultValue Tuple.first
    in
    H.select
        [ HE.onInput (toMsg << valueFromLabel) ]
        (List.map optionForTuple valuesWithLabels)



---- Wait page-related functions.


waitPageLink : String -> String -> String
waitPageLink eventurl proxyurl =
    API.buildUrl [ "wait" ]
        [ UB.string "event" eventurl
        , UB.string "jump" proxyurl
        ]


openWaitPage : String -> String -> Cmd msg
openWaitPage eventurl proxyurl =
    Ports.openNewWindowPort (waitPageLink eventurl proxyurl)



-- Misc.


openLogsButton : msg -> H.Html msg
openLogsButton msg =
    let
        config =
            { action = SendMsg msg
            , bgColor = "blue"
            , fgColor = "white"
            , isActive = True
            , isPending = False
            , style = Full "fas fa-history"
            , text = "Logs"
            }
    in
    buttonCreator config


breadcrumb : List ( String, String ) -> H.Html msg -> H.Html msg
breadcrumb parents currentPage =
    let
        parentElements =
            List.map
                (\( path, text ) ->
                    H.a
                        [ HA.href path
                        , HA.class "font-bold hover:underline"
                        ]
                        [ H.text text ]
                )
                parents
    in
    parentElements
        ++ [ currentPage ]
        |> List.map (List.singleton >> H.li [ HA.class "inline" ])
        |> H.ul [ HA.class "breadcrumb list-none truncate" ]
