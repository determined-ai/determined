module Components.LabelWidget exposing
    ( Config
    , LabelEdit(..)
    , Mode(..)
    , State
    , init
    , view
    )

import Browser.Dom as Dom
import Html as H
import Html.Attributes as HA
import Html.Events as HE
import Json.Decode as D
import Set exposing (Set)
import Task


type LabelEdit
    = Add Int String -- Int = experiment ID, String = label
    | Remove Int String


type alias Config msg =
    { toMsg : State -> Maybe (Cmd msg) -> msg
    , toLabelEditMsg : State -> LabelEdit -> msg
    , toLabelClickedMsg : String -> msg
    , displayAtMost : Maybe Int
    , maxLabelLength : Int
    }


type Mode
    = AbridgedDisplay
    | FullDisplay
    | Editing


type alias State =
    { mode : Mode
    , id : Int
    , displayEditButton : Bool
    , newLabelValue : String
    }


init : Int -> State
init id =
    { mode = AbridgedDisplay
    , id = id
    , displayEditButton = False
    , newLabelValue = ""
    }


getDisplayableLabels : Maybe Int -> Set String -> Set String
getDisplayableLabels displayAtMost labels =
    case displayAtMost of
        Just n ->
            Set.toList labels
                |> List.take n
                |> Set.fromList

        Nothing ->
            labels


renderBadge : Config msg -> Maybe (String -> msg) -> String -> H.Html msg
renderBadge config maybeRemoveMsg label =
    let
        classes =
            HA.class
                "mr-1 mt-1 px-1 text-xs bg-orange-200 border border-gray-500 rounded flex-grow-0 max-w-xs"

        styles =
            HA.style "overflow-wrap" "break-word"

        title =
            HA.title label

        maybeTruncated =
            if String.length label > config.maxLabelLength then
                String.slice 0 config.maxLabelLength label ++ "..."

            else
                label
    in
    case maybeRemoveMsg of
        Just removeMsg ->
            H.span [ classes, styles, title ]
                [ H.div []
                    [ H.text maybeTruncated
                    , H.i
                        [ HA.class "fas fa-times pl-1 text-red-700 cursor-pointer"
                        , HE.onClick (removeMsg label)
                        ]
                        []
                    ]
                ]

        Nothing ->
            H.span
                [ classes
                , styles
                , title
                , HA.class "cursor-pointer"
                , HE.onClick (config.toLabelClickedMsg label)
                ]
                [ H.text maybeTruncated ]


renderBadges : Config msg -> Maybe (String -> msg) -> Set String -> List (H.Html msg)
renderBadges config maybeRemoveMsg labels =
    Set.toList labels
        |> List.map (renderBadge config maybeRemoveMsg)


ellipsis : Config msg -> State -> H.Html msg
ellipsis config state =
    let
        newState =
            { state | mode = FullDisplay }

        handler =
            HE.onClick (config.toMsg newState Nothing)
    in
    H.span [ HA.class "mr-1 mt-1 px-1 hover:shadow cursor-pointer border rounded", handler ]
        [ H.text "..." ]


onEnter : msg -> H.Attribute msg
onEnter msg =
    let
        isEnter code =
            if code == 13 then
                D.succeed msg

            else
                D.fail "not ENTER"
    in
    HE.on "keydown" (D.andThen isEnter HE.keyCode)


view : Config msg -> State -> Set String -> H.Html msg
view config state labels =
    let
        contents =
            case state.mode of
                AbridgedDisplay ->
                    let
                        displayable =
                            getDisplayableLabels config.displayAtMost labels

                        badges =
                            renderBadges config Nothing displayable

                        more =
                            Set.size displayable < Set.size labels

                        editClasses =
                            "fas fa-edit text-xs align-middle text-gray-500 hover:text-black cursor-pointer absolute left-0"

                        editClickHandler =
                            let
                                newState =
                                    { state | mode = Editing }

                                cmd =
                                    "input_"
                                        ++ String.fromInt state.id
                                        |> focusInput (config.toMsg newState Nothing)
                                        |> Just
                            in
                            HE.onClick (config.toMsg newState cmd)

                        edit =
                            H.div [ HA.class "relative" ]
                                [ H.div []
                                    [ H.div [ HA.style "transform" "translateY(-25%)" ]
                                        [ H.i [ HA.class editClasses, editClickHandler ] [] ]
                                    ]
                                ]

                        elements =
                            if more then
                                badges ++ [ ellipsis config state ]

                            else
                                badges

                        maybeWithEditButton =
                            if state.displayEditButton then
                                elements ++ [ edit ]

                            else
                                elements

                        -- Ensure non-zero height.
                        zeroWidthSpace =
                            "\u{200B}"
                    in
                    H.div [ HA.class "flex flex-row" ] (maybeWithEditButton ++ [ H.text zeroWidthSpace ])

                FullDisplay ->
                    let
                        finishedEditingMsg =
                            config.toMsg { state | mode = AbridgedDisplay } Nothing

                        done =
                            H.div
                                [ HA.class "ml-1 mt-1 text-xs px-1 cursor-pointer hover:shadow rounded absolute top-0 right-0"
                                , HA.style "transform" "translate(75%,-50%)"
                                , HE.onClick finishedEditingMsg
                                ]
                                [ H.i [ HA.class "fas fa-times" ] [] ]

                        badges =
                            renderBadges config Nothing labels ++ [ done ]
                    in
                    H.div [ HA.class "flex flex-wrap flex-row max-w-xs relative" ] badges

                Editing ->
                    let
                        badgeRemoveMsg =
                            config.toLabelEditMsg state << Remove state.id

                        badges =
                            renderBadges config (Just badgeRemoveMsg) labels

                        enterPressedMsg =
                            config.toLabelEditMsg
                                state
                                (Add state.id state.newLabelValue)

                        inputUpdatedMsg =
                            \x -> config.toMsg { state | newLabelValue = x } Nothing

                        edit =
                            H.div [ HA.class "relative" ]
                                [ H.input
                                    [ HA.placeholder "New label"
                                    , HA.id ("input_" ++ String.fromInt state.id)
                                    , onEnter enterPressedMsg
                                    , HE.onInput inputUpdatedMsg
                                    , HA.autofocus True
                                    , HA.class "border border-gray-500 rounded w-16 mt-1 text-xs px-1"
                                    ]
                                    []
                                ]

                        finishedEditingMsg =
                            config.toMsg
                                { state | mode = AbridgedDisplay, newLabelValue = "" }
                                Nothing

                        done =
                            H.div
                                [ HA.class "ml-1 mt-1 text-xs px-1 cursor-pointer hover:shadow rounded absolute top-0 right-0"
                                , HA.style "transform" "translate(75%,-50%)"
                                , HE.onClick finishedEditingMsg
                                ]
                                [ H.i [ HA.class "fas fa-times" ] [] ]
                    in
                    H.div [ HA.class "flex flex-wrap flex-row max-w-xs relative" ]
                        (badges ++ [ edit, done ])

        mouseEnterHandler =
            config.toMsg { state | displayEditButton = True } Nothing

        mouseLeaveHandler =
            config.toMsg { state | displayEditButton = False } Nothing
    in
    H.div [ HE.onMouseEnter mouseEnterHandler, HE.onMouseLeave mouseLeaveHandler ]
        [ contents ]


focusInput : msg -> String -> Cmd msg
focusInput msg elementID =
    Task.attempt (\_ -> msg) (Dom.focus elementID)
