module View exposing (viewBody)

import Css
import Css.Global
import Html exposing (Html, a, div, i, img, nav, span, text)
import Html.Attributes as HA exposing (attribute, class, href, id, src, style, tabindex)
import Html.Events as HE
import Html.Styled
import Model exposing (Model, Page(..))
import Msg exposing (Msg)
import Page.Cluster
import Page.CommandList
import Page.Common
import Page.ExperimentDetail
import Page.ExperimentList
import Page.LogViewer
import Page.NotebookList
import Page.ShellList
import Page.TensorBoardList
import Page.TrialDetail
import Route
import String.Extra
import Toast
import View.SlotChart


viewBody : Model -> List (Html Msg)
viewBody model =
    let
        ( maybeTopBar, maybeSideMenu ) =
            case ( model.page, model.session.user ) of
                ( _, Just _ ) ->
                    ( topAppBar model, renderSideMenu model )

                ( _, _ ) ->
                    ( text "", text "" )
    in
    [ globalStyles
    , div [ class "flex flex-col min-h-screen max-h-screen bg-white" ]
        [ div [ class "flex-shrink" ]
            [ maybeTopBar ]
        , div
            [ class "flex flex-row flex-1 overflow-y-hidden" ]
            [ maybeSideMenu

            -- Provide an ID for the focus hack (see Main.elm). The tabindex is necessary to allow the
            -- element to take focus on Chrome.
            , div
                [ id "det-main-container"
                , tabindex -1
                , class "outline-none h-full w-full overflow-y-scroll relative px-8"
                ]
                [ viewContent model ]
            ]
        , div [ class "flex-shrink" ]
            [ renderToasts model.toasts ]
        , maybeCriticalErrorPopup model
        , maybeUpdateVersionPopup model
        ]
    ]


viewContent : Model -> Html Msg
viewContent model =
    case model.page of
        Init ->
            text ""

        NotFound ->
            span [ class "p-4" ] [ text "Page not found!" ]

        Cluster pageModel ->
            Page.Cluster.view pageModel model.session
                |> Html.map Msg.ClusterMsg

        CommandList pageModel ->
            Page.CommandList.view pageModel model.session
                |> Html.map Msg.CommandListMsg

        ExperimentDetail pageModel ->
            Page.ExperimentDetail.view pageModel model.session
                |> Html.map Msg.ExperimentDetailMsg

        ExperimentList pageModel ->
            Page.ExperimentList.view pageModel model.session
                |> Html.map Msg.ExperimentListMsg

        ShellList pageModel ->
            Page.ShellList.view pageModel model.session
                |> Html.map Msg.ShellListMsg

        NotebookList pageModel ->
            Page.NotebookList.view pageModel model.session
                |> Html.map Msg.NotebookListMsg

        TensorBoardList pageModel ->
            Page.TensorBoardList.view pageModel model.session
                |> Html.map Msg.TensorBoardListMsg

        TrialDetail pageModel ->
            Page.TrialDetail.view pageModel model.session
                |> Html.map Msg.TrialDetailMsg

        LogViewer pageModel ->
            Page.LogViewer.view pageModel model.session
                |> Html.map Msg.LogViewerMsg


globalStyles : Html msg
globalStyles =
    Css.Global.global
        [ -- The html and body styles are needed to make the footer stick to the bottom.
          Css.Global.html
            [ Css.height (Css.pct 100)
            , Css.overflow Css.hidden
            ]
        , Css.Global.body
            [ Css.height (Css.pct 100)
            , Css.overflow Css.hidden
            , Css.displayFlex
            , Css.flexDirection Css.column
            ]

        -- A flex style missing from tailwind, needed for the content to make the footer stick to
        -- the bottom.
        , Css.Global.class "flex-1-0-auto"
            [ Css.flexGrow (Css.int 1)
            , Css.flexShrink (Css.int 0)
            , Css.flexBasis Css.auto
            ]

        -- dAI specific colors, taken from the main website.
        , Css.Global.class "bg-blue-dai"
            [ Css.backgroundColor (Css.hex "#0e1e2b")
            ]
        , Css.Global.class "bg-orange-dai"
            [ Css.backgroundColor (Css.hex "#f77b21")
            ]
        ]
        |> Html.Styled.toUnstyled


docsLink : Html Msg
docsLink =
    div
        [ HA.style "font-size" "12px"
        , HA.style "color" "rgb(221, 221, 221)"
        , HA.style "padding" "11px 0"
        , HA.style "margin-left" "36px"
        , HA.style "border-bottom" "4px solid rgba(0, 0, 0, 0)"
        , HA.style "border-top" "4px solid rgba(0, 0, 0, 0)"
        ]
        [ a
            [ class "no-underline hover:text-white flex items-center"
            , HA.target "_blank"
            , href "/docs/"
            ]
            [ text <| "Docs "
            , i
                [ class "icon-popout"
                , attribute "style" "font-size: 16px; padding-left: 4px;"
                ]
                []
            ]
        ]


apiDocsLink : Html Msg
apiDocsLink =
    div
        [ HA.style "font-size" "12px"
        , HA.style "color" "rgb(221, 221, 221)"
        , HA.style "padding" "11px 0"
        , HA.style "margin-left" "36px"
        , HA.style "border-bottom" "4px solid rgba(0, 0, 0, 0)"
        , HA.style "border-top" "4px solid rgba(0, 0, 0, 0)"
        ]
        [ a
            [ class "no-underline hover:text-white flex items-center"
            , HA.target "_blank"
            , href "/swagger-ui/"
            ]
            [ text <| "API "
            , i
                [ class "icon-popout"
                , attribute "style" "font-size: 16px; padding-left: 4px;"
                ]
                []
            ]
        ]


currentUserDisplay : Model -> Html Msg
currentUserDisplay model =
    let
        onClickClose =
            Msg.ToggleUserDropdownMenu False |> HE.onClick

        overlay =
            div
                [ class "fixed inset-0 opacity-0 z-10"
                , onClickClose
                ]
                []

        menuClasses =
            "cursor-pointer z-20 absolute top-0 right-0 bg-white rounded "
                ++ "flex items-center hover:text-blue-400"
                |> class

        menu =
            case model.session.user of
                Just _ ->
                    let
                        attributes =
                            [ "font-family: -apple-system, BlinkMacSystemFont, "
                                ++ "'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', "
                                ++ "'Microsoft YaHei', 'Helvetica Neue', Helvetica, "
                                ++ "Arial, sans-serif, 'Apple Color Emoji', "
                                ++ "'Segoe UI Emoji', 'Segoe UI Symbol';"
                            , "font-size: 14px;"
                            , "color: rgba(0, 0, 0, 0.65);"
                            , "padding: 5px 12px;"
                            , "line-height: 1;"
                            ]
                    in
                    div [ class "relative" ]
                        [ div
                            [ menuClasses
                            , attribute "style" "margin-top: 4px; padding: 4px 0; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);"
                            ]
                            [ a
                                [ onClickClose
                                , Route.toString Route.Logout |> href
                                , class "whitespace-no-wrap"
                                , attribute "style" (String.join " " attributes)
                                ]
                                [ text "Sign Out" ]
                            ]
                        ]

                Nothing ->
                    text ""

        onClick =
            model.userDropdownOpen
                |> not
                |> Msg.ToggleUserDropdownMenu
                |> HE.onClick

        button user =
            div
                [ onClick
                , id "avatar"
                , class
                    ("cursor-pointer text-white hover:text-white font-bold"
                        ++ " rounded-full flex items-center justify-center h-6 w-6"
                    )
                , style "margin-left" "36px"
                , style "font-size" "10px"
                , style "background-color" "hsl(184,53%,50%)"
                ]
                [ String.left 1 user.username |> String.Extra.toTitleCase |> text ]
    in
    case ( model.userDropdownOpen, model.session.user ) of
        ( False, Just { user } ) ->
            button user

        ( True, Just { user } ) ->
            div []
                [ overlay
                , button user
                , menu
                ]

        ( _, Nothing ) ->
            text ""


topAppBar : Model -> Html Msg
topAppBar model =
    let
        -- We use breakpoints to make the cluster usage widget disappear from the top app bar
        -- when the browser viewport hits 768px.
        maybeClusterText =
            Maybe.andThen
                (\slots ->
                    (case slots of
                        [] ->
                            " No Agents"

                        someSlots ->
                            View.SlotChart.allocationPercent someSlots
                    )
                        |> text
                        |> List.singleton
                        |> span
                            [ HA.style "margin-left" "8px"
                            , HA.style "font-size" "12px"
                            ]
                        |> Just
                )
                model.slots

        maybeClusterView =
            case model.session.user of
                Just _ ->
                    a
                        [ class "flex text-gray-300 hover:text-white items-center"
                        , href (Route.toString Route.Cluster)
                        ]
                        [ i [ class "icon-cluster text-xl" ] []
                        , Maybe.withDefault (text "")
                            maybeClusterText
                        ]

                Nothing ->
                    text ""
    in
    nav
        [ class "flex flex-row items-center justify-between bg-no-repeat bg-cover bg-top px-4"
        , attribute "style" "background-color: #0D1E2B"
        , HA.style "font-family" "'Objektiv Mk3', Arial, Helvetica, sans-serif"
        ]
        [ -- Left side: product logo/name.
          a [ href "/" ]
            [ img
                [ class "self-center h-8"
                , attribute "style" "width: 128px; height: 20px;"
                , src "/public/images/logo-on-dark-horizontal.svg"
                ]
                []
            ]

        -- Separate the left and right sides.
        , div [ style "flex-grow" "1" ]
            []

        -- Right side.
        , maybeClusterView
        , apiDocsLink
        , docsLink
        , currentUserDisplay model
        ]


sideTabs : Model -> Html Msg
sideTabs model =
    let
        iconAttribute =
            attribute "style" "font-size: 20px; width: 20px;"

        tabs =
            [ ( text "Dashboard", Route.Dashboard, i [ class "icon-user flex-grow-0", iconAttribute ] [] )
            , ( text "Experiments", Route.ExperimentListReact, i [ class "icon-experiment flex-grow-0", iconAttribute ] [] )
            , ( text "Tasks", Route.TaskList, i [ class "icon-tasks flex-grow-0", iconAttribute ] [] )
            , ( text "Cluster", Route.Cluster, i [ class "icon-cluster flex-grow-0", iconAttribute ] [] )
            ]

        selectedTabRoute =
            case model.page of
                Cluster _ ->
                    Just Route.Cluster

                CommandList _ ->
                    Just (Route.CommandList Route.defaultCommandLikeListOptions)

                ExperimentDetail _ ->
                    Just (Route.ExperimentList Route.defaultExperimentListOptions)

                ExperimentList _ ->
                    Just (Route.ExperimentList Route.defaultExperimentListOptions)

                NotebookList _ ->
                    Just (Route.NotebookList Route.defaultCommandLikeListOptions)

                TensorBoardList _ ->
                    Just (Route.TensorBoardList Route.defaultCommandLikeListOptions)

                ShellList _ ->
                    Just (Route.ShellList Route.defaultCommandLikeListOptions)

                TrialDetail _ ->
                    Just (Route.ExperimentList Route.defaultExperimentListOptions)

                _ ->
                    Nothing

        tab ( body, route, icon ) =
            [ a
                [ class
                    ("flex flex-row items-center py-1 pl-3 pr-4 border-l-4 no-underline "
                        ++ "hover:text-blue-400 text-sm outline-none "
                        ++ (if selectedTabRoute == Just route then
                                "text-blue-400 border-blue-400 border-solid"

                            else
                                "text-gray-700 border-transparent"
                           )
                    )
                , style "margin-bottom" "16px"
                , href (Route.toString route)
                ]
                [ icon
                , div
                    [ style "font-size" "12px"
                    , style "font-family" "'Objektiv Mk3', Arial, Helvetica, sans-serif"
                    , style "margin-left" "16px"
                    , style "letter-spacing" "0"
                    , class "font-light"
                    ]
                    [ body ]
                ]
            ]
    in
    List.concatMap tab tabs |> div [ class "flex flex-col" ]


footerTabs : Html Msg
footerTabs =
    let
        iconAttribute =
            attribute "style" "font-size: 20px; width: 20px;"
    in
    div
        [ class "flex flex-col" ]
        [ a
            [ class
                ("flex flex-row items-center py-1 pl-3 pr-4 border-l-4 no-underline "
                    ++ "hover:text-blue-400 text-sm outline-none "
                    ++ "text-gray-700 border-transparent"
                )
            , style "margin-bottom" "16px"
            , HA.target "_blank"
            , href "/det/logs"
            ]
            [ i [ class "icon-logs flex-grow-0", iconAttribute ] []
            , div
                [ style "font-size" "12px"
                , style "font-family" "'Objektiv Mk3', Arial, Helvetica, sans-serif"
                , style "margin-left" "16px"
                , style "letter-spacing" "0"
                , class "font-light"
                ]
                [ text "Master Logs" ]
            ]
        ]


sideBarFooter : Model -> Html Msg
sideBarFooter model =
    div
        [ class "flex flex-col" ]
        [ footerTabs
        , div
            [ class "flex justify-center"
            , style "background-color" "#ececec"
            , style "color" "#666"
            , style "font-size" "10px"
            , style "padding" "6px 0"
            ]
            [ text ("Version " ++ model.version) ]
        ]


renderToast : Toast.Toast -> Html Msg
renderToast toast =
    div
        [ class "max-w-md p-2 bg-gray-600 mb-4 rounded border shadow text-orange-300 font-bold px-12 relative text-base" ]
        [ text toast.message
        , div [ class "absolute right-0 top-0" ]
            [ i
                [ class "fas fa-times hover:shadow cursor-pointer mr-1"
                , HE.onClick (Msg.ToastExpired toast.id)
                ]
                []
            ]
        ]


renderToasts : List Toast.Toast -> Html Msg
renderToasts messages =
    div
        [ class "absolute top-0 left-0 w-screen flex flex-row justify-center"
        , style "transform" "translateY(-100%)"
        ]
        [ div [ class "flex flex-col" ]
            (List.map renderToast messages)
        ]


{-|

    maybeCriticalErrorPopup renders a critical message in a popup that grays
    out the background.  The user cannot close the popup.

-}
maybeCriticalErrorPopup : Model -> Html Msg
maybeCriticalErrorPopup model =
    case model.criticalError of
        Just message ->
            div [ class "fixed inset-0" ]
                [ div [ class "fixed inset-0 bg-gray-900 opacity-50" ] []
                , div [ class "fixed inset-0" ]
                    [ div [ class "container flex mx-auto" ]
                        [ div [ class "w-1/3" ] []
                        , div [ class "w-1/3" ]
                            [ div [ class "border rounded w-full shadow mt-32 bg-white" ]
                                [ div [ class "text-xl bold bg-red-300 rounded-t border-b" ]
                                    [ div [ class "text-center p-4" ]
                                        [ div [ class "text-5xl" ]
                                            [ i [ class "far fa-frown" ] [] ]
                                        , div [] [ text "Unknown error!" ]
                                        ]
                                    ]
                                , div [ class "p-4" ] [ text message ]
                                ]
                            ]
                        , div [ class "w-1/3" ] []
                        ]
                    ]
                ]

        Nothing ->
            text ""


maybeUpdateVersionPopup : Model -> Html Msg
maybeUpdateVersionPopup model =
    case model.info of
        Just info ->
            let
                buttonUpdateNow =
                    Page.Common.buttonCreator
                        { action = Page.Common.SendMsg Msg.OutdatedVersion
                        , bgColor = "blue"
                        , fgColor = "white"
                        , isActive = True
                        , isPending = False
                        , style = Page.Common.TextOnly
                        , text = "Update Now"
                        }
            in
            if info.version /= model.version then
                div
                    [ class "absolute flex flex-col bg-white bottom-0 right-0 m-6 p-6"
                    , style "box-shadow" "0 3px 6px -4px rgba(0, 0, 0, 0.12), 0 6px 16px 0 rgba(0, 0, 0, 0.08), 0 9px 28px 8px rgba(0, 0, 0, 0.05)"
                    ]
                    [ div [ class "mb-2" ] [ text "New WebUI Version" ]
                    , span [ class "mb-2" ]
                        [ text "WebUI version "
                        , i [ class "font-bold" ] [ text info.version ]
                        , text " is available."
                        ]
                    , buttonUpdateNow
                    ]

            else
                text ""

        Nothing ->
            text ""


renderSideMenu : Model -> Html Msg
renderSideMenu model =
    div
        [ HA.id "side-menu"
        , class "flex flex-col justify-between"
        , style "padding" "16px 0 0 0"
        , style "background-color" "#f7f7f7"
        , style "border-right" "solid 1px #dddddd"
        ]
        [ sideTabs model, sideBarFooter model ]
