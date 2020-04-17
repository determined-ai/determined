module View.Modal exposing
    ( Config
    , contentView
    , titleClasses
    , view
    )

import Html as H
import Html.Attributes as HA
import Page.Common


{-| Configuration for a modal.
-}
type alias Config msg =
    { content : H.Html msg
    , attributes : List (H.Attribute msg)
    , closeMsg : msg
    }


type alias Content msg =
    { header : H.Html msg
    , body : H.Html msg
    , footer : Maybe (H.Html msg)
    , buttons : List (H.Html msg)
    }


titleClasses : String
titleClasses =
    "text-2xl"


contentView : Content msg -> H.Html msg
contentView content =
    [ H.div [ HA.class "pl-4 p-2 flex-shrink" ]
        [ content.header ]
    , H.div
        [ HA.class "flex-grow pl-4 p-2 overflow-auto"
        ]
        [ content.body ]
    ]
        ++ (case content.footer of
                Just footer ->
                    [ H.div [ HA.class "flex-shrink" ] [ footer ] ]

                Nothing ->
                    []
           )
        ++ (if List.length content.buttons > 0 then
                [ H.div [ HA.class "flex-shrink p-2" ]
                    [ H.div [ HA.class "flex flex-row justify-end" ]
                        [ Page.Common.horizontalList content.buttons
                        ]
                    ]
                ]

            else
                []
           )
        |> H.div [ HA.class "flex flex-col w-full" ]


view : Config msg -> H.Html msg
view conf =
    let
        closeButton =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg conf.closeMsg
                , bgColor = "transparent"
                , fgColor = "black"
                , isActive = True
                , isPending = False
                , style = Page.Common.IconOnly "fas fa-times text-black"
                , text = "Close"
                }

        content =
            [ H.div [ HA.class "absolute top-0 right-0 p-3 z-50" ] [ closeButton ]
            , conf.content
            ]
    in
    H.div [ HA.class "fixed inset-0 z-40", HA.style "overflow" "hidden" ]
        [ H.div [ HA.class "fixed inset-0 bg-gray-600 opacity-75 z-40" ] []
        , H.div [ HA.class "p-1 fixed inset-0 flex justify-center items-center z-50" ]
            [ H.div
                (conf.attributes
                    ++ [ HA.class "flex p-2 bg-white rounded shadow modal relative"
                       , HA.style "min-width" "25rem"
                       , HA.style "max-width" "80vw"
                       , HA.style "max-height" "80vh"
                       , HA.style "overflow" "hidden"
                       ]
                )
                content
            ]
        ]
