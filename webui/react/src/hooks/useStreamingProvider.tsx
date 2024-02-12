import React, { createContext } from "react";
import useWebSocket from 'react-use-websocket';

type UserStreamingContext = {
    sendMessage: (message: string, keep?: boolean) => void;
    readyState: number;
    lastMessage: WebSocketEventMap['message'] | null,
  };
  
  export const Streaming = createContext<UserStreamingContext>({
    sendMessage: () => {},
    readyState: -1,
    lastMessage: null,
  });

  export const StreamingProvider: React.FC<React.PropsWithChildren> = ({ children }) => {

    const socketUrl = 'ws://localhost:8080/stream';

        const {
        sendMessage,
        readyState,
        lastMessage,
        } = useWebSocket(socketUrl, {
        onOpen: () => console.log('websocket opened!'),
        //Will attempt to reconnect on all close events, such as server shutting down
        shouldReconnect: () => true,
        });

    return (
        <Streaming.Provider
          value={{
            sendMessage,
            readyState,
            lastMessage
          }}>
          {children}
        </Streaming.Provider>
    )
  }