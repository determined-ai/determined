module Components.Logs exposing
    ( Config
    , Model
    , Msg(..)
    , defaultPollInterval
    , init
    , subscriptions
    , update
    , view
    )

{-| A component for displaying sequences of textual logs.
-}

import API
import Browser.Dom as Dom
import Communication as Comm
import Html as H
import Html.Attributes as HA
import Html.Events as HE
import Html.Keyed
import Html.Lazy
import Json.Decode as D
import Maybe.Extra
import Page.Common
import Ports
import Result.Extra
import Task
import Time



---- Type definitions.


type Msg log
    = NoOp
    | Tick
    | DoFullscreen Bool
    | DoCopyToClipboard
    | DoJumpToBottom
    | FullscreenChanged Bool
    | Scrolled ScrollInfo
    | GotHeight Float
    | GotLogs Bool (List log)
      -- Errors.
    | GotCriticalError Comm.SystemError
    | GotAPIError Bool API.APIError


{-| For either the head or the tail of the logs, whether a request for more data is currently out
and being waited for (Pending), all data on that end have been fetched and we should not make any
more requests (Finished), or neither (Inactive).
-}
type RequestState
    = Inactive
    | Pending
    | Finished


{-| The configuration options for an instance of this component.

`scrollId` and `containerId` are two arbitrary unique HTML IDs to assign to elements within the
view.

Due to an unfortunate leaky abstraction in the way that lazy HTML works, the `getText` element
should always refer to the same actual JavaScript object across calls, or performance will suffer
for large logs. (The issue is that two distinct non-primitive objects can never compare equal, even
if they are identically defined functions or structurally identical objects, which breaks the
laziness check.)

-}
type alias Config log msg =
    { toMsg : Msg log -> msg
    , pollInterval : Float
    , scrollId : String
    , containerId : String
    , getId : log -> Int
    , getText : log -> String
    , keepPolling : Bool
    , poll :
        API.RequestHandlers (Msg log) (List log)
        ->
            { greaterThanId : Maybe Int
            , lessThanId : Maybe Int
            , tailLimit : Maybe Int
            }
        -> Cmd (Msg log)
    }


type alias ScrollInfo =
    { offsetHeight : Float
    , scrollHeight : Float
    , scrollTop : Float
    }


type alias Model log =
    { data :
        -- The `entries` field contains a list of lists of the log entries that have been fetched.
        -- For display, the messages for all entries are concatenated together in the natural order.
        -- This is a performance optimization: storing the entries in a flat list means the browser
        -- gets quite bogged down by the time we have hundreds of thousands of rows, even with the
        -- use of Html.Keyed and Html.Lazy. Chunking them means we can do much less work later on.
        Maybe
            { entries : List (List log)
            , minId : Int
            , maxId : Int
            }
    , bottomed : Bool
    , lastScroll : Float
    , lastHeight : Float
    , fullscreen : Bool
    , headState : RequestState
    , tailState : RequestState
    }



---- Constants and helpers.


{-| How many log messages to request on page load.
-}
initialLogsFetch : Int
initialLogsFetch =
    10000


{-| How many additional log messages to request each time the user scrolls to the top of the view.
-}
extraLogsFetch : Int
extraLogsFetch =
    5000


defaultPollInterval : Float
defaultPollInterval =
    1000


{-| The fullscreen change event's `target.ownerDocument.fullscreenElement` attribute contains either
an element or `null`; this decoder extracts only whether or not it is null.
-}
fullscreenDecoder : (Bool -> msg) -> D.Decoder msg
fullscreenDecoder tagger =
    [ D.null False
    , D.succeed True
    ]
        |> D.oneOf
        |> D.at [ "target", "ownerDocument", "fullscreenElement" ]
        |> D.map tagger


scrollInfoDecoder : (ScrollInfo -> msg) -> D.Decoder msg
scrollInfoDecoder tagger =
    D.map3 ScrollInfo
        (D.at [ "target", "offsetHeight" ] D.float)
        (D.at [ "target", "scrollHeight" ] D.float)
        (D.at [ "target", "scrollTop" ] D.float)
        |> D.map tagger


{-| Find the lowest and highest values that a function takes on the elements of a nonempty list.
-}
bounds : (a -> comparable) -> a -> List a -> ( comparable, comparable )
bounds f x =
    List.foldl
        (\a ( curMin, curMax ) ->
            let
                next =
                    f a
            in
            ( min next curMin, max next curMax )
        )
        ( f x, f x )


requestHandlers : Bool -> API.RequestHandlers (Msg logs) (List logs)
requestHandlers isTail =
    { onSuccess = GotLogs isTail
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIError isTail
    }


pollTailCmd : Config log msg -> Model log -> Cmd (Msg log)
pollTailCmd config model =
    config.poll (requestHandlers True) <|
        case model.data of
            Nothing ->
                { greaterThanId = Nothing
                , lessThanId = Nothing
                , tailLimit = Just initialLogsFetch
                }

            Just data ->
                { greaterThanId = Just data.maxId
                , lessThanId = Nothing
                , tailLimit = Nothing
                }



---- Initialization.


init : Config log msg -> ( Model log, Cmd (Msg log) )
init config =
    let
        model =
            { data = Nothing
            , bottomed = True
            , lastScroll = 0
            , lastHeight = 0
            , fullscreen = False
            , headState = Inactive
            , tailState = Pending
            }
    in
    ( model, pollTailCmd config model )



---- Update.


update : Config log msg -> Msg log -> Model log -> ( Model log, Cmd (Msg log), Maybe Comm.SystemError )
update config msg model =
    case msg of
        NoOp ->
            ( model, Cmd.none, Nothing )

        Tick ->
            if model.tailState == Inactive then
                ( { model | tailState = Pending }, pollTailCmd config model, Nothing )

            else
                ( model, Cmd.none, Nothing )

        DoFullscreen f ->
            ( model
            , -- In addition to requesting fullscreen, request focus on the scrolling element so
              -- that keyboard scrolling works immediately after entering fullscreen.
              if f then
                Cmd.batch
                    [ Ports.requestFullscreenPort config.containerId
                    , Task.attempt (always NoOp) (Dom.focus config.scrollId)
                    ]

              else
                Ports.exitFullscreenPort ()
            , Nothing
            )

        DoCopyToClipboard ->
            ( model
            , Ports.copyToClipboard config.scrollId
            , Nothing
            )

        FullscreenChanged f ->
            ( { model | fullscreen = f }
            , -- When the log view exits fullscreen, the top remains at the same place while the
              -- height decreases, which would register as exiting bottomed-out state; avoid that by
              -- requesting a jump to the bottom.
              if not f && model.bottomed then
                Ports.jumpToPointPort ( config.scrollId, 0 )

              else
                Cmd.none
            , Nothing
            )

        DoJumpToBottom ->
            ( model, Ports.jumpToPointPort ( config.scrollId, 0 ), Nothing )

        Scrolled info ->
            let
                scrollTopMax =
                    info.scrollHeight - info.offsetHeight

                -- Subtract 1 because there seem to be rounding issues that can make this miss
                -- situations that should count as bottoming out.
                bottomed =
                    (info.scrollTop >= scrollTopMax - 1)
                        || (model.bottomed && info.scrollTop >= model.lastScroll)

                needHeadRequest =
                    info.scrollTop < 50 && model.headState == Inactive

                getHead =
                    if needHeadRequest then
                        config.poll (requestHandlers False)
                            { greaterThanId = Nothing
                            , lessThanId = Maybe.map .minId model.data
                            , tailLimit = Just extraLogsFetch
                            }

                    else
                        Cmd.none

                headState =
                    if needHeadRequest then
                        Pending

                    else
                        model.headState
            in
            ( { model
                | lastScroll = info.scrollTop
                , lastHeight = info.scrollHeight
                , bottomed = bottomed
                , headState = headState
              }
            , getHead
            , Nothing
            )

        GotHeight height ->
            ( { model | lastHeight = height }, Cmd.none, Nothing )

        GotLogs isTail [] ->
            let
                -- On a tail request, getting an empty response just means there are no new logs, so
                -- we should keep polling. On a head request, it means we've gotten all the way to
                -- the beginning of the logs, so make sure not to do any more head requests.
                ( headState, tailState ) =
                    if isTail then
                        ( model.headState, Inactive )

                    else
                        ( Finished, model.tailState )
            in
            ( { model
                | headState = headState
                , tailState = tailState
              }
            , Cmd.none
            , Nothing
            )

        GotLogs isTail ((e :: es) as newEntries) ->
            let
                ( minNewId, maxNewId ) =
                    bounds config.getId e es

                ( minId, maxId, entries ) =
                    case model.data of
                        Nothing ->
                            ( minNewId, maxNewId, [ newEntries ] )

                        Just data ->
                            ( min minNewId data.minId
                            , max maxNewId data.maxId
                            , if isTail then
                                data.entries ++ [ newEntries ]

                              else
                                newEntries :: data.entries
                            )

                ( headState, tailState ) =
                    if isTail then
                        ( model.headState, Inactive )

                    else
                        ( Inactive, model.tailState )
            in
            ( { model
                | data =
                    Just { entries = entries, minId = minId, maxId = maxId }
                , headState = headState
                , tailState = tailState
              }
            , if model.bottomed then
                Ports.jumpToPointPort ( config.scrollId, 0 )

              else if not isTail then
                -- Prepending items would normally cause the visible content to scroll down; this
                -- command causes the element to instead maintain its position relative to its
                -- bottom.
                Ports.jumpToPointPort
                    ( config.scrollId
                    , model.lastHeight - model.lastScroll
                    )

              else
                -- When we jump to the bottom or prepend something, we get a scroll event containing
                -- an updated height (which we need to know for the next time we prepend items).
                -- However, appending items while not at the bottom changes the height without
                -- triggering a scroll, so explicitly ask for the new height in this case.
                Dom.getViewportOf config.scrollId
                    |> Task.attempt
                        (Result.Extra.unwrap
                            NoOp
                            (.scene >> .height >> GotHeight)
                        )
            , Nothing
            )

        GotCriticalError e ->
            ( model, Cmd.none, Just e )

        GotAPIError isTail err ->
            let
                _ =
                    -- TODO(jgevirtz): Report error to user.
                    Debug.log "Failed to get logs" err

                ( headState, tailState ) =
                    if isTail then
                        ( model.headState, Inactive )

                    else
                        ( Inactive, model.tailState )
            in
            ( { model | headState = headState, tailState = tailState }, Cmd.none, Nothing )



---- View.


concatView : (log -> String) -> List log -> H.Html (Msg log)
concatView getText =
    List.map getText
        >> String.concat
        >> H.text


containerView : Config log msg -> List (List log) -> H.Html (Msg log)
containerView config entries =
    let
        body =
            List.map
                (\e ->
                    ( e |> List.map config.getId |> List.minimum |> Maybe.Extra.unwrap "" String.fromInt
                    , Html.Lazy.lazy2 concatView config.getText e
                    )
                )
                entries
    in
    H.div
        [ HA.class "h-full bg-white" ]
        [ Html.Keyed.node "pre"
            [ HA.class "text-xs overflow-auto h-full outline-none p-2"

            -- The ID identifies the element in JS so we can request fullscreen and jump to the
            -- bottom.
            , HA.id config.scrollId

            -- We track scrolling to know whether the element is bottomed out or not.
            , HE.on "scroll" (scrollInfoDecoder Scrolled)

            -- The tabindex allows the element to be focused and scrolled by keyboard.
            , HA.tabindex 0
            ]
            body
        ]


view : Config log msg -> Model log -> List (H.Html (Msg log)) -> H.Html msg
view config model extraButtons =
    let
        body =
            case model.data of
                Nothing ->
                    [ Page.Common.centeredLoadingWidget ]

                Just { entries } ->
                    let
                        fullscreenButton =
                            Page.Common.buttonCreator
                                { action = Page.Common.SendMsg (DoFullscreen (not model.fullscreen))
                                , bgColor = "blue"
                                , fgColor = "white"
                                , isActive = True
                                , isPending = False
                                , style =
                                    if model.fullscreen then
                                        Page.Common.IconOnly "fas fa-compress"

                                    else
                                        Page.Common.IconOnly "fas fa-expand"
                                , text =
                                    if model.fullscreen then
                                        "Exit full screen (Esc)"

                                    else
                                        "Full screen"
                                }

                        copyToClipboardButton =
                            Page.Common.buttonCreator
                                { action = Page.Common.SendMsg DoCopyToClipboard
                                , bgColor = "blue"
                                , fgColor = "white"
                                , isActive = True
                                , isPending = False
                                , style = Page.Common.IconOnly "far fa-copy"
                                , text = "Copy to clipboard"
                                }

                        bottomButton =
                            if model.bottomed then
                                H.text ""

                            else
                                Page.Common.buttonCreator
                                    { action = Page.Common.SendMsg DoJumpToBottom
                                    , bgColor = "blue"
                                    , fgColor = "white"
                                    , isActive = True
                                    , isPending = False
                                    , style = Page.Common.TextOnly
                                    , text = "Jump to bottom"
                                    }

                        headSpinner =
                            case model.headState of
                                Pending ->
                                    H.div
                                        [ HA.class "pt-3 absolute inset-x-0 flex justify-center"
                                        , HA.style "transform" "scale(2)"
                                        ]
                                        [ Page.Common.spinner ]

                                _ ->
                                    H.text ""

                        buttons =
                            H.div
                                [ HA.class "absolute opacity-25 hover:opacity-100 smooth-opacity"
                                , HA.style "top" "20px"
                                , HA.style "right" "30px"
                                ]
                                [ bottomButton
                                    :: extraButtons
                                    ++ [ copyToClipboardButton, fullscreenButton ]
                                    |> Page.Common.horizontalList
                                ]
                    in
                    [ H.div
                        [ HA.id config.containerId
                        , HA.class "relative h-full"
                        , HE.on "fullscreenchange" (fullscreenDecoder FullscreenChanged)
                        ]
                        [ headSpinner
                        , buttons
                        , containerView config entries
                        ]
                    ]
    in
    H.div [ HA.class "relative border border-gray-300", HA.style "height" "100%" ] body
        |> H.map config.toMsg



---- Subscriptions.


subscriptions : Config log msg -> Model log -> Sub (Msg log)
subscriptions config _ =
    if config.keepPolling then
        Time.every config.pollInterval (\_ -> Tick)

    else
        Sub.none
