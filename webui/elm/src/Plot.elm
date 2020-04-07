module Plot exposing (Config, Model, Msg, PlotScale(..), init, scaleSelectorView, update, view)

{-| This is a module for creating line plots of the style we want for Determined. The code design is
inspired by `elm-sortable-table`: like that module, this one does not keep track of the data to
display. Instead, the data are passed directly to the view function and only used there, while the
model stores only ancillary information.
-}

import Axis
import Color
import Formatting
import Html as H exposing (div)
import Html.Attributes as HA
import Html.Events as HE
import Json.Decode as D
import Page.Common
import Path exposing (Path)
import Scale
import Shape
import SubPath
import TypedSvg as TS exposing (g, svg)
import TypedSvg.Attributes as SA exposing (class, fill, stroke, transform)
import TypedSvg.Attributes.InPx as InPx
import TypedSvg.Core exposing (Svg)
import TypedSvg.Events as TSE
import TypedSvg.Types as TST exposing (Fill(..), Length(..), Transform(..))



---- External configuration interface.


type alias Config data msg =
    { tooltip : data -> H.Html msg
    , toMsg : Msg data -> msg
    , getX : data -> Float
    , getY : data -> Float
    , xLabel : String
    , yLabel : String
    , title : String
    }



---- Internal type definitions.


{-| A point in input coordinates.
-}
type alias Loc =
    ( Float, Float )


type alias Padding =
    { top : Float
    , right : Float
    , bottom : Float
    , left : Float
    }


type alias Size =
    { width : Float
    , height : Float
    }



---- External type definitions and other non-view things.


type PlotScale
    = Linear
    | Log Float


type alias Model data =
    { hovered : Maybe data
    , size : Maybe Size
    , scale : PlotScale
    }


type Msg data
    = Hover data
    | Unhover
    | Resize Size
    | SetScale PlotScale


init : Model data
init =
    { hovered = Nothing, size = Nothing, scale = Linear }


update : Msg data -> Model data -> Model data
update msg model =
    case msg of
        Hover d ->
            { model | hovered = Just d }

        Unhover ->
            { model | hovered = Nothing }

        Resize s ->
            let
                _ =
                    Debug.log "resize" s

                size =
                    -- Attempting to draw a plot with zero size just produces something malformed,
                    -- so go back to drawing nothing if we get a zero-size update.
                    if s.width > 0 && s.height > 0 then
                        Just s

                    else
                        Nothing
            in
            { model | size = size }

        SetScale scale ->
            { model | scale = scale }



---- Miscellaneous visual parameters. They could be configurable, but for now we keep them constant
---- for simplicity.


plotScalesWithLabels : List ( PlotScale, String )
plotScalesWithLabels =
    [ ( Linear, "linear" )
    , ( Log 2, "log2" )
    , ( Log 10, "log10" )
    ]


{-| The space in pixels between the plot's data area and the edges of the SVG element.
-}
padding : Padding
padding =
    { top = 30
    , right = 20
    , bottom = 40
    , left = 75
    }


{-| The configuration defining the shape of the dots for data points.
-}
dotConfig : Shape.Arc
dotConfig =
    { innerRadius = 0.0
    , outerRadius = 3.0
    , cornerRadius = 0.0
    , startAngle = 0.0
    , endAngle = 7.0
    , padAngle = 0.0
    , padRadius = 0.0
    }


{-| The configuration defining the shape of the invisible dots used to trigger mouseover events.
-}
hoverDotConfig : Shape.Arc
hoverDotConfig =
    { dotConfig | outerRadius = 7.0 }


{-| The color for data dots that are not being hovered.
-}
defaultColor : Color.Color
defaultColor =
    Color.rgb 0.3 0.7 1


{-| The color for data dots that are being hovered.
-}
hoveredColor : Color.Color
hoveredColor =
    Color.rgb 0.1 1 0.1


{-| The color for the lines connecting the dots.
-}
lineColor : Color.Color
lineColor =
    defaultColor


gridColor : Color.Color
gridColor =
    Color.rgba 0 0 0 0.1



---- View functions.


tickFormat : Scale.ContinuousScale Float -> PlotScale -> Int -> Float -> String
tickFormat scale plotScale ticks x =
    let
        mag =
            abs x

        showTick =
            case plotScale of
                Linear ->
                    True

                -- For log scales with bases greater than 2, the ticks can be spaced unevenly if
                -- there are multiple ticks per power of the base, getting closer together as the
                -- values approach a power from below. This code reproduces some logic from
                -- elm-visualization to hide labels in log scales when the ticks get close together.
                Log base ->
                    let
                        sig0 =
                            mag / (base ^ toFloat (round (logBase base mag)))

                        sig =
                            if sig0 * base < base - 0.5 then
                                sig0 * base

                            else
                                sig0

                        maxSig =
                            max 1 (base * toFloat ticks / toFloat (List.length (Scale.ticks scale 10)))
                    in
                    sig <= maxSig
    in
    if x == 0 then
        "0"

    else if not showTick then
        ""

    else
        Formatting.compactFloat 3 3 x


xAxis : Int -> Scale.ContinuousScale Float -> Svg msg
xAxis ticks xScale =
    Axis.bottom [ Axis.tickCount ticks ] xScale


yAxis : Int -> Scale.ContinuousScale Float -> (Float -> String) -> Svg msg
yAxis ticks yScale yTickFormat =
    Axis.left [ Axis.tickCount ticks, Axis.tickFormat yTickFormat ] yScale


line : Scale.ContinuousScale Float -> Scale.ContinuousScale Float -> List Loc -> Path
line xScale yScale =
    List.map (\( x, y ) -> Just ( Scale.convert xScale x, Scale.convert yScale y ))
        >> Shape.line Shape.linearCurve


dots : Scale.ContinuousScale Float -> Scale.ContinuousScale Float -> Shape.Arc -> List ( data, Loc ) -> List ( data, Path )
dots xScale yScale config =
    let
        makeDot ( x, y ) =
            List.map
                (SubPath.translate ( Scale.convert xScale x, Scale.convert yScale y ))
                (Shape.arc config)
    in
    List.map <| Tuple.mapSecond <| makeDot


gridLine : Float -> Float -> Float -> Float -> Svg msg
gridLine x0 y0 x1 y1 =
    TS.line
        [ SA.x1 (TST.Px x0)
        , SA.x2 (TST.Px x1)
        , SA.y1 (TST.Px y0)
        , SA.y2 (TST.Px y1)
        , stroke gridColor
        ]
        []


bounds : List Float -> ( Float, Float )
bounds xs =
    case xs of
        [] ->
            ( 0, 0 )

        [ x ] ->
            ( x - 0.5, x + 0.5 )

        _ ->
            ( List.minimum xs |> Maybe.withDefault 0, List.maximum xs |> Maybe.withDefault 0 )


scaleSelectorView : (Msg data -> msg) -> H.Html msg
scaleSelectorView toMsg =
    Page.Common.selectFromValues plotScalesWithLabels
        Linear
        (SetScale >> toMsg)


{-| A container that positions a child element of unknown width so that its center is at a given
horizontal offset relative to the container's own center, with the position clamped so that the
child does not extend horizontally beyond the bounds of the container. Since the child's width is
unknown, this can't be done by straightforward calculation of positions; instead, we place the child
in a flex row and give it two neighbors whose relative sizes are adjusted to push it into the right
position without letting it go outside the container.
-}
offsetClampWrapper : Float -> H.Html msg -> H.Html msg
offsetClampWrapper offset child =
    let
        -- If the offset is positive (to the right), the left neighbor should be bigger by twice the
        -- offset, so we set the left neighbor's size to that value and the right neighbor's size to
        -- zero. If it is negative, the left neighbor should be smaller, but flex positioning
        -- doesn't deal with negative sizes, so we have to switch to making the left one have zero
        -- size and the right one bigger.
        offsetLeft =
            max (2 * offset) 0

        offsetRight =
            max (-2 * offset) 0
    in
    H.div [ HA.class "flex flex-row w-full h-0" ]
        [ H.div [ HA.style "flex" ("1 1 " ++ String.fromFloat offsetLeft ++ "px") ] []
        , H.div [ HA.style "flex" "0 0 auto", HA.style "max-width" "100%" ] [ child ]
        , H.div [ HA.style "flex" ("1 1 " ++ String.fromFloat offsetRight ++ "px") ] []
        ]


plotView : Config data msg -> Model data -> Size -> List data -> List (H.Html msg)
plotView config model sz data =
    let
        -- Input-space data bounds.
        xs =
            List.map config.getX data

        ys =
            List.map config.getY data

        ( x0, x1 ) =
            bounds xs

        ( y0, y1 ) =
            bounds ys

        -- Pixel-space data bounds.
        cx0 =
            0

        cx1 =
            sz.width - padding.left - padding.right

        cy0 =
            sz.height - padding.top - padding.bottom

        cy1 =
            0

        -- Scales.
        ( xTicks, yTicks ) =
            if List.length data < 2 then
                ( List.length data, List.length data )

            else
                ( round <| min (x1 - x0 + 1) ((cx1 - cx0) / 40), round <| (cy0 - cy1) / 30 )

        xScale =
            Scale.linear ( cx0, cx1 ) ( x0, x1 )
                |> (if List.length data < 2 then
                        identity

                    else
                        Scale.nice xTicks
                   )

        yScale =
            case model.scale of
                Linear ->
                    Scale.linear ( cy0, cy1 ) ( y0, y1 )
                        |> (if List.length data < 2 then
                                identity

                            else
                                Scale.nice yTicks
                           )

                Log base ->
                    -- TODO: We would like to apply `nice` to logarithmic scales as well, but that's
                    -- buggy, so avoid it for now.
                    Scale.log base ( cy0, cy1 ) ( y0, y1 )

        -- Axes and labels.
        yTickFormat =
            tickFormat yScale model.scale yTicks

        axes =
            [ g [ transform [ Translate (padding.left - 1) (sz.height - padding.bottom) ] ]
                [ xAxis xTicks xScale ]
            , g [ transform [ Translate (padding.left - 1) padding.top ] ]
                [ yAxis yTicks yScale yTickFormat ]
            ]

        axisLabels =
            [ TS.text_
                [ SA.fontWeight TST.FontWeightBold
                , transform [ Translate (padding.left + (cx0 + cx1) / 2) sz.height ]
                , SA.textAnchor TST.AnchorMiddle
                , SA.dominantBaseline TST.DominantBaselineTextAfterEdge
                ]
                [ H.text config.xLabel ]
            , TS.text_
                [ SA.fontWeight TST.FontWeightBold
                , transform [ Translate 0 (padding.top + (cy0 + cy1) / 2), Rotate 270 0 0 ]
                , SA.textAnchor TST.AnchorMiddle
                , SA.dominantBaseline TST.DominantBaselineTextBeforeEdge
                ]
                [ H.text config.yLabel ]
            ]

        title =
            [ g [ transform [ Translate (padding.left + (cx0 + cx1) / 2) 0 ] ]
                [ TS.text_
                    [ SA.fontWeight TST.FontWeightBold
                    , SA.fontSize (Percent 150)
                    , SA.textAnchor TST.AnchorMiddle
                    , SA.dominantBaseline TST.DominantBaselineTextBeforeEdge
                    ]
                    [ H.text config.title ]
                ]
            ]

        -- Grid lines.
        xTickVals =
            Scale.ticks xScale xTicks |> List.map (Scale.convert xScale)

        yTickVals =
            Scale.ticks yScale yTicks |> List.map (Scale.convert yScale)

        grid =
            [ g [ transform [ Translate padding.left padding.top ] ]
                (List.map (\x -> gridLine x cy0 x cy1) xTickVals
                    ++ List.map (\y -> gridLine cx0 y cx1 y) yTickVals
                )
            ]

        -- The actual data in the plot.
        dataAndLocs =
            List.map (\d -> ( d, ( config.getX d, config.getY d ) )) data

        dataLines =
            [ g [ transform [ Translate padding.left padding.top ], class [ "series" ] ]
                [ Path.element (line xScale yScale (List.map Tuple.second dataAndLocs))
                    [ stroke lineColor
                    , InPx.strokeWidth 2
                    , fill FillNone
                    ]
                ]
            ]

        dataDots =
            [ g [ transform [ Translate padding.left padding.top ] ]
                (dots xScale yScale dotConfig dataAndLocs
                    |> List.map
                        (\( d, p ) ->
                            let
                                clr =
                                    if Just d == model.hovered then
                                        hoveredColor

                                    else
                                        defaultColor
                            in
                            Path.element p [ fill <| Fill <| clr ]
                        )
                )
            ]

        -- The invisible elements used to trigger mouseover events; they are larger than the visible
        -- dots to make it easier to mouse over them without having too much visual clutter. A full
        -- solution might look for the closest data point to the mouse, but for now we just put
        -- these dots down and let them overlap as they may.
        hoverDataDots =
            [ g [ transform [ Translate padding.left padding.top ] ]
                (dots xScale yScale hoverDotConfig dataAndLocs
                    |> List.map
                        (\( d, p ) ->
                            Path.element p
                                [ fill <| Fill <| Color.rgba 0 0 0 0
                                , TSE.onMouseEnter (Hover d)
                                , TSE.onMouseLeave Unhover
                                ]
                        )
                )
            ]

        -- Tooltip.
        makeTooltip datum =
            let
                body =
                    config.tooltip datum

                x =
                    padding.left + Scale.convert xScale (config.getX datum)

                y =
                    padding.top + Scale.convert yScale (config.getY datum) - sz.height + 20
            in
            offsetClampWrapper (x - sz.width / 2)
                (H.div
                    [ HA.class "bg-orange-400 p-2 relative"
                    , HA.style "top" <| String.fromFloat y ++ "px"
                    ]
                    [ body ]
                )

        tooltip =
            model.hovered
                |> Maybe.map makeTooltip
                |> Maybe.withDefault (H.text "")

        -- In SVG, z-ordering is determined by physical order in the document; accordingly, these
        -- are roughly in increasing order by importance.
        allElems =
            List.concat [ axes, axisLabels, title, dataLines, dataDots, grid, hoverDataDots ]
    in
    [ H.map config.toMsg <| svg [ SA.height (Percent 100), SA.width (Percent 100) ] allElems
    , tooltip
    ]


{-| A custom element that generates resize events when its size changes. See
`webui/public/js/determined-shims.js` for the JS-side custom element definition.

IMPORTANT! To allow for the chart to fill into the resize container,
the parent container MUST have the `relative` class name.

-}
resizeContainer : (Msg data -> msg) -> List (H.Html msg) -> H.Html msg
resizeContainer toMsg =
    H.node "resize-monitor"
        [ HA.class "block absolute w-full h-full"
        , HE.on "resize"
            (D.map2 (\w h -> toMsg (Resize (Size w h)))
                (D.at [ "detail", "width" ] D.float)
                (D.at [ "detail", "height" ] D.float)
            )
        ]


{-| The top-level view for users of this module.
-}
view : Config data msg -> Model data -> List data -> H.Html msg
view config model data =
    -- If we haven't received a size yet, render an empty resize monitor container to get the first
    -- size notification. Once we get that, render the plot.
    resizeContainer config.toMsg <|
        case model.size of
            Nothing ->
                []

            Just sz ->
                plotView config model sz data
