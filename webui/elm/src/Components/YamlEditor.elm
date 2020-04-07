module Components.YamlEditor exposing
    ( Config
    , ContentUpdate
    , State
    , destroy
    , init
    , resize
    , subscriptions
    , view
    )

import Html as H
import Html.Attributes as HA
import Ports


type alias ContentUpdate =
    Ports.AceContentUpdate


type alias Config msg =
    { newContentToMsg : ContentUpdate -> msg
    , containerIdOverride : Maybe String
    }


type alias State =
    { initialValue : String
    , containerId : String
    }


init : Config msg -> String -> ( State, Cmd msg )
init config initialValue =
    let
        containerId =
            case config.containerIdOverride of
                Just s ->
                    s

                Nothing ->
                    "editor"

        command =
            Ports.setUpAceEditor ( containerId, initialValue )

        state =
            { initialValue = initialValue
            , containerId = containerId
            }
    in
    ( state, command )


resize : State -> Cmd msg
resize state =
    Ports.resizeAceEditor state.containerId


destroy : State -> Cmd msg
destroy state =
    Ports.destroyAceEditor state.containerId


subscriptions : Config msg -> Sub msg
subscriptions config =
    Sub.batch
        [ Ports.aceContentUpdated config.newContentToMsg
        ]


view : State -> H.Html msg
view state =
    H.div
        [ HA.id state.containerId, HA.class "absolute inset-0" ]
        []
