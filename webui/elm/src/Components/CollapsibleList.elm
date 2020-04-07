module Components.CollapsibleList exposing
    ( Model
    , Msg
    , init
    , update
    , view
    )

import Html as H
import Html.Attributes as HA
import Page.Common


type alias Model =
    { isCollapsed : Bool
    , limit : Int
    }


type Msg
    = Toggle



---- Init


init : Int -> Model
init limit =
    { isCollapsed = True, limit = limit }



---- Update


update : Msg -> Model -> Model
update msg model =
    case msg of
        Toggle ->
            { model | isCollapsed = not model.isCollapsed }



---- View


view : Model -> (Msg -> msg) -> List (H.Html msg) -> List (H.Html msg)
view model toMsg items =
    let
        needsCollapsing =
            List.length items > model.limit

        itemsToShow =
            if model.isCollapsed then
                List.take model.limit items

            else
                items

        toggleButton =
            if model.isCollapsed then
                H.span [ HA.title "Show more" ] [ H.text "..." ]

            else
                H.span [ HA.class "text-xs", HA.title "Show less" ] [ H.text "(hide)" ]
    in
    itemsToShow
        ++ (if needsCollapsing then
                [ H.button [ Page.Common.onClickStopPropagation <| toMsg Toggle ]
                    [ toggleButton ]
                ]

            else
                [ H.text "" ]
           )
