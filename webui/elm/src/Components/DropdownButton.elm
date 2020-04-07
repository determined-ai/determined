module Components.DropdownButton exposing
    ( Config
    , Model
    , Msg
    , OutMsg(..)
    , SelectOption
    , init
    , update
    , view
    )

import Html
import Html.Attributes
import Html.Events


type alias SelectOption msg =
    { action : msg
    , label : String
    }


{-| `Config` is used for static stuff.

primaryButton: The message to fire in the parent when selecting the primary button, and its text.
selectOptions: Top-to-bottom, the message to fire in the parent when selecting an option in the
dropdown, and its text.
bgColor: The Tailwind-recognized English name for the primary button's background color.
TODO: Support all Tailwind colors as focus colors; see tailwind.config.js.
fgColor: The Tailwind-recognized English name for the primary button's text color.

-}
type alias Config msg =
    { primaryButton : SelectOption msg
    , selectOptions : List (SelectOption msg)
    , bgColor : String
    , fgColor : String
    }


type Model
    = Model Bool


type Msg msg
    = ParentMsg msg
    | Toggle


type OutMsg msg
    = OptionSelected msg


init : Model
init =
    Model False


isOpen : Model -> Bool
isOpen (Model open) =
    open


update : Msg msg -> Model -> ( Model, Maybe (OutMsg msg) )
update msg model =
    case msg of
        ParentMsg actionMsg ->
            ( Model False
            , Just (OptionSelected actionMsg)
            )

        Toggle ->
            ( Model (not (isOpen model))
            , Nothing
            )


view : Config msg -> Model -> Html.Html (Msg msg)
view { primaryButton, selectOptions, bgColor, fgColor } model =
    Html.div
        []
        (splitButtonView model primaryButton bgColor fgColor
            ++ (if isOpen model then
                    [ selectView selectOptions
                    , overlay Toggle
                    ]

                else
                    [ Html.text "" ]
               )
        )


splitButtonView : Model -> SelectOption msg -> String -> String -> List (Html.Html (Msg msg))
splitButtonView model { action, label } bgColor fgColor =
    let
        buttonClassStrings =
            [ "font-bold" -- Style text.
            , "py-1" -- Set up spacing.
            , "smooth-opacity" -- Set up opacity animation.
            , "focus:outline-none focus:shadow-outline" -- Set up nicely visible focus outlines.
            , "hover:shadow focus:shadow" -- Button-y outlines.
            , "bg-" ++ bgColor ++ "-500" -- Apply color customization.
            , "hover:bg-" ++ bgColor ++ "-700"
            , "focus:shadow-outline-" ++ bgColor
            , "text-" ++ fgColor
            , "z-30 relative" -- Make this clickable even when overlay is open.
            , "focus:z-20" -- Put the focus shadow on this element behind its sibling.
            ]

        leftButtonClasses =
            Html.Attributes.class
                (String.join " " (buttonClassStrings ++ [ "rounded-l", "px-2" ]))

        rightButtonClasses =
            Html.Attributes.class
                (String.join " "
                    (buttonClassStrings
                        ++ [ "rounded-r", "px-1-5" ]
                        ++ (if isOpen model then
                                [ "bg-" ++ bgColor ++ "-700" ]

                            else
                                []
                           )
                    )
                )
    in
    [ Html.button
        [ leftButtonClasses
        , Html.Events.onClick (ParentMsg action)
        ]
        [ Html.text label ]
    , Html.button
        [ rightButtonClasses
        , Html.Events.onClick Toggle
        ]
        [ Html.i [ Html.Attributes.class "fas fa-caret-down" ] [] ]
    ]


selectView : List (SelectOption msg) -> Html.Html (Msg msg)
selectView options =
    let
        enclosingDivClass =
            Html.Attributes.class "relative"

        dropdownDivClass =
            Html.Attributes.class
                """
                z-20 border border-gray-700 shadow absolute top-0 mt-1 rounded overflow-y-auto 
                overflow-x-hidden
                """
    in
    Html.div
        [ enclosingDivClass ]
        [ Html.div
            [ dropdownDivClass ]
            (List.map optionView options)
        ]


optionView : SelectOption msg -> Html.Html (Msg msg)
optionView { action, label } =
    let
        itemClass =
            Html.Attributes.class
                """
                flex flex-row py-1 px-2 border-b cursor-pointer hover:bg-orange-200 bg-white 
                w-full
                """
    in
    Html.button
        [ itemClass
        , Html.Events.onClick (ParentMsg action)
        ]
        [ Html.text label ]


overlay : msg -> Html.Html msg
overlay msg =
    Html.div
        [ Html.Attributes.class "fixed inset-0 w-full h-full z-10"
        , Html.Events.onClick msg
        ]
        []
