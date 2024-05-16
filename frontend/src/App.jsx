import React, { useState, useEffect } from "react";
import Wrapper from "./components/Wrapper";
import Header from "./components/Header";
import Footer from "./components/Footer";

import Player from "@madzadev/audio-player";
import "@madzadev/audio-player/dist/index.css";

import { PrismLight as SyntaxHighlighter } from "react-syntax-highlighter";

import bash from "react-syntax-highlighter/dist/esm/languages/prism/bash";
import jsx from "react-syntax-highlighter/dist/esm/languages/prism/jsx";
import javascript from "react-syntax-highlighter/dist/esm/languages/prism/javascript";
SyntaxHighlighter.registerLanguage("bash", bash);
SyntaxHighlighter.registerLanguage("jsx", jsx);
SyntaxHighlighter.registerLanguage("javascript", javascript);

let url = "";

// TODO Player requires a trackList with at least 3 elements
// TODO Player remove time / slider for infinite audio

const defaultTrackList = [
  {
    url: url + "/v1/playlist/trending",
    title: "trending",
    tags: ["trending"],
  },
  {
    url: url + "/v1/playlist/top",
    title: "top",
    tags: ["top"],
  },
  {
    url: url + "/v1/playlist/weekly",
    title: "weekly",
    tags: ["weekly"],
  },
  {
    url: url + "/v1/playlist/monthly",
    title: "monthly",
    tags: ["monthly"],
  },
];

const App = () => {
  const [trackList, setTrackList] = useState(defaultTrackList);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(url + "/v1/playlist");
        if (!response.ok) {
          throw new Error("Failed to fetch data");
        }

        const rjson = await response.json();

        let data = [];

        // default sort
        for (let i = 0; i < defaultTrackList.length; i++) {
          for (let playlist in rjson) {
            if (defaultTrackList[i].title == playlist) {
              data.push({
                url: url + "/v1/playlist/" + playlist,
                title: playlist,
                tags: [playlist],
              });
              delete rjson[playlist];
            }
          }
        }

        for (let playlist in rjson) {
          data.push({
            url: url + "/v1/playlist/" + playlist,
            title: playlist,
            tags: [playlist],
          });
        }

        console.log("data", data, defaultTrackList);

        setTrackList(data);
      } catch (error) {
        console.error("Error fetching data:", error);
      }
    };

    fetchData();
  }, []);

  return (
    <Wrapper>
      <Header />
      <Player
        trackList={trackList}
        sortTracks={false}
        includeTags={false}
        includeSearch={false}
        autoPlayNextTrack={false}
      />
      <Footer />
    </Wrapper>
  );
};

export default App;
