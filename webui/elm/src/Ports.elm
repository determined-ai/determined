port module Ports exposing
    ( AceContentUpdate
    , aceContentUpdated
    , copiedToClipboard
    , copyToClipboard
    , destroyAceEditor
    , exitFullscreenPort
    , jumpToPointPort
    , kickResizePort
    , loadAnalytics
    , openNewWindowPort
    , requestFullscreenPort
    , resizeAceEditor
    , setAnalyticsEventPort
    , setAnalyticsIdentityPort
    , setAnalyticsPagePort
    , setPageTitle
    , setUpAceEditor
    )

{-| All ports for the WebUI.
-}


{-| Virtual DOM workaround -- see comments in index.html.
-}
port kickResizePort : () -> Cmd msg


{-| Open the given URL in a new window.
-}
port openNewWindowPort : String -> Cmd msg


{-| Request that the element with the given ID be made fullscreen.
-}
port requestFullscreenPort : String -> Cmd msg


{-| Request that the document leave fullscreen mode.
-}
port exitFullscreenPort : () -> Cmd msg


{-| Request that the element with the given ID scroll so that its top is the given distance above
its bottom.
-}
port jumpToPointPort : ( String, Float ) -> Cmd msg



-- Ace editor ports for fancy YAML editing.


port destroyAceEditor : String -> Cmd msg


port setUpAceEditor : ( String, String ) -> Cmd msg


port resizeAceEditor : String -> Cmd msg


type alias AceContentUpdate =
    { content : String
    , badSyntax : Bool
    }


port aceContentUpdated : (AceContentUpdate -> msg) -> Sub msg



-- Clipboard


port copyToClipboard : String -> Cmd msg


port copiedToClipboard : (Bool -> msg) -> Sub msg



-- Page


port setPageTitle : String -> Cmd a



-- Segment Analytics


port loadAnalytics : String -> Cmd msg


port setAnalyticsIdentityPort : String -> Cmd msg


port setAnalyticsEventPort : ( String, String ) -> Cmd msg


port setAnalyticsPagePort : String -> Cmd msg
