module Modals.ShowConfiguration exposing
    ( Model
    , Msg
    , init
    , openConfig
    , subscriptions
    , update
    , view
    )

{-| A component for showing experiment configuration in a modal.
-}

import Browser.Events
import Html as H
import Html.Attributes as HA
import Json.Decode as D
import Json.Encode as E
import Page.Common
import Ports
import Types
import View.Modal as Modal
import Yaml.Encode as YE


type Model
    = Closed
    | OpenConfig Types.ExperimentConfig


type Msg
    = CloseModal
    | DoCopyToClipboard


openConfig : Types.ExperimentConfig -> ( Model, Cmd Msg )
openConfig config =
    ( OpenConfig config, Cmd.none )


init : Model
init =
    Closed


subscriptions : Sub Msg
subscriptions =
    Browser.Events.onKeyUp
        (D.field "key" D.string
            |> D.andThen
                (\key ->
                    if key == "Escape" then
                        D.succeed CloseModal

                    else
                        D.fail "not Escape"
                )
        )


update : Model -> Msg -> ( Model, Cmd Msg )
update model msg =
    case msg of
        CloseModal ->
            ( Closed, Cmd.none )

        DoCopyToClipboard ->
            ( model, Ports.copyToClipboard "show-config-content" )


renderModal : H.Html Msg -> H.Html Msg
renderModal c =
    Modal.view
        { content = c
        , attributes = []
        , closeMsg = CloseModal
        }


viewConfig : Types.ExperimentConfig -> H.Html Msg
viewConfig config =
    let
        configAsString =
            E.dict identity identity config
                |> YE.encode

        copyToClipboardButton =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg DoCopyToClipboard
                , bgColor = "white"
                , fgColor = "blue"
                , isActive = True
                , isPending = False
                , style = Page.Common.IconOnly "far fa-copy"
                , text = "Copy to clipboard"
                }
    in
    Modal.contentView
        { header =
            H.span [ HA.class Modal.titleClasses ]
                [ H.span [] [ H.text "Configuration" ]
                , H.span [ HA.class "text-base" ] [ copyToClipboardButton ]
                ]
        , body =
            H.div [ HA.class "mb-4 overflow-auto border border-gray-300" ]
                [ H.pre
                    [ HA.class "p-4 text-xs"
                    , HA.style "height" "50vh"
                    , HA.id "show-config-content"
                    ]
                    [ H.text configAsString ]
                ]
        , footer = Nothing
        , buttons = []
        }


view : Model -> H.Html Msg
view model =
    case model of
        OpenConfig config ->
            renderModal (viewConfig config)

        Closed ->
            H.text ""
