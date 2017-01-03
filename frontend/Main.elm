import Html exposing (Html, button, div, text, ul, li, img, input, form)
import Html.Events exposing (onClick, on)
import Html.Attributes exposing (class, src, action, method, href, target, type_)
import Http
import Json.Decode as Decode
import FileReader
import Debug


main =
  Html.program { init = init, view = view, update = update, subscriptions = subscriptions }


-- MODEL

type alias Model =
    {
        cardName : String,
        details : CardDetails
    }

type alias CardDetails =
    {
        pictures : List Picture
    }

type alias Picture =
    {
        cloud_id : String,
        web_url : String
    }

type alias Files =
    List FileReader.NativeFile


init = (Model "" (CardDetails []), getCardName)

-- UPDATE

type Msg = NewCard (Result Http.Error String)
         | NewCardDetails (Result Http.Error CardDetails)
         | FileUpload Files
         | UploadComplete (Result Http.Error Decode.Value)

update : Msg -> Model -> (Model, Cmd Msg)
update msg model =
  let
      _ = Debug.log "update" (msg, model)
  in
      case msg of
          NewCard (Ok cardName) ->
              let
                  newModel = { model | cardName = cardName}
              in
                  (newModel, getCardDetails newModel)

          -- Ignore errors for now
          NewCard (Err _) ->
              ({ model | cardName = (toString (Err))}, Cmd.none)

          NewCardDetails (Ok details) ->
              ({ model | details = details }, Cmd.none)

          NewCardDetails (Err _) ->
              (model, Cmd.none)

          FileUpload (files) ->
              case List.head files of
                  Just file ->
                      (model, sendFileToServer model file)
                  Nothing ->
                      (model, Cmd.none)


          UploadComplete (Ok _) ->
              (model, getCardDetails model)

          UploadComplete (Err _) ->
              (model, Cmd.none)


-- VIEW

view : Model -> Html Msg
view model =
  div []
    [ div [] [ Html.h2 [] [ text model.cardName ] ]
    , Html.a [ Html.Attributes.href ("/v1/make_pdf/" ++ model.cardName),
             Html.Attributes.target "_blank" ] [ text "Download PDF" ]
    , ul [ class "picture-list" ] <| List.map viewPicture model.details.pictures
    , input [ type_ "file", onchange FileUpload ] []
    ]

viewPicture picture =
    li [] [ text (toString picture.cloud_id)
          , img [ src picture.web_url ] [] ]

onchange action =
    on
        "change"
        (Decode.map action FileReader.parseSelectedFiles)

-- SUBSCRIPTIONS

subscriptions model =
    Sub.none

-- HTTP

getCardName =
    let
        url = "/v1/make_new_card"
    in
        Http.send NewCard (Http.get url decodeCardName)

decodeCardName =
    Decode.at ["name"] Decode.string

getCardDetails model =
    let
        url = "/v1/get_card/" ++ model.cardName
    in
        Http.send NewCardDetails (Http.get url decodeCardDetails)

decodeCardDetails : Decode.Decoder CardDetails
decodeCardDetails =
    Decode.at ["pictures"] (Decode.map CardDetails (Decode.list pictureDecoder))

pictureDecoder : Decode.Decoder Picture
pictureDecoder =
    Decode.map2 Picture
        (Decode.field "cloud_id" Decode.string)
        (Decode.field "web_url" Decode.string)

sendFileToServer : Model -> FileReader.NativeFile -> Cmd Msg
sendFileToServer model buf =
    let
        body =
            Http.multipartBody
                [ FileReader.filePart "file" buf ]
    in
        Http.post ("/v1/add_picture/" ++ model.cardName) body Decode.value
            |> Http.send UploadComplete
