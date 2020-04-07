module Components.AdvancedButton exposing
    ( Model
    , Msg
    , init
    , update
    , view
    )

{-| A component for creating stateful buttons.
-}

import Html as H
import Html.Attributes as HA
import Page.Common
import Ports
import Task
import View.Modal as Modal


type alias Model msg =
    { needsConfirmation : Bool
    , confirmBtnText : String
    , dismissBtnText : String
    , promptOpen : Bool
    , promptTitle : String
    , promptText : String
    , config : Page.Common.ButtonConfig msg
    }


type Msg
    = Confirm
    | Cancel
    | Prompt



---- Constants and helpers


init : Bool -> String -> Page.Common.ButtonConfig msg -> Model msg
init needsConfirmation promptText config =
    { needsConfirmation = needsConfirmation
    , confirmBtnText = "Confirm"
    , dismissBtnText = "Dismiss"
    , promptOpen = False
    , promptText = promptText
    , promptTitle = "Confirm Action"
    , config = config
    }


update : Msg -> Model msg -> ( Model msg, Cmd msg )
update msg model =
    let
        msgToCmd aMsg =
            Task.succeed aMsg
                |> Task.perform identity
    in
    case msg of
        Confirm ->
            let
                newModel =
                    { model | promptOpen = False }
            in
            case model.config.action of
                Page.Common.SendMsg actionMsg ->
                    ( newModel, msgToCmd actionMsg )

                Page.Common.OpenUrl _ url ->
                    ( newModel, Ports.openNewWindowPort url )

        Prompt ->
            ( { model | promptOpen = True }, Cmd.none )

        Cancel ->
            ( { model | promptOpen = False }, Cmd.none )



---- View


promptView : Model msg -> (Msg -> msg) -> H.Html msg
promptView model toMsg =
    if model.promptOpen then
        let
            title =
                H.span [ HA.class Modal.titleClasses ] [ H.text model.promptTitle ]

            body =
                H.p [] [ H.text model.promptText ]

            confirmButton =
                Page.Common.buttonCreator
                    { action = Page.Common.SendMsg (toMsg Confirm)
                    , bgColor = "red"
                    , fgColor = "white"
                    , isActive = True
                    , isPending = False
                    , style = Page.Common.TextOnly
                    , text = model.confirmBtnText
                    }

            dismissButton =
                Page.Common.buttonCreator
                    { action = Page.Common.SendMsg (toMsg Cancel)
                    , bgColor = "blue"
                    , fgColor = "white"
                    , isActive = True
                    , isPending = False
                    , style = Page.Common.TextOnly
                    , text = model.dismissBtnText
                    }
        in
        Modal.view
            { content =
                Modal.contentView
                    { header = title
                    , body = body
                    , footer = Nothing
                    , buttons = [ dismissButton, confirmButton ]
                    }
            , attributes = [ HA.style "min-width" "30rem" ]
            , closeMsg = toMsg Cancel
            }

    else
        H.text ""


buttonView : Model msg -> (Msg -> msg) -> H.Html msg
buttonView model toMsg =
    let
        originalConf =
            model.config

        updatedConf =
            if model.needsConfirmation then
                { originalConf | action = Page.Common.SendMsg (toMsg Prompt) }

            else
                { originalConf | action = Page.Common.SendMsg <| toMsg <| Confirm }
    in
    Page.Common.buttonCreator updatedConf


view : Model msg -> (Msg -> msg) -> H.Html msg
view model toMsg =
    H.div [ HA.class "inline-block" ]
        [ buttonView model toMsg
        , promptView model toMsg
        ]
