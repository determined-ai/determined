import React, {
  createContext,
  MutableRefObject,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import uPlot from 'uplot';

interface SyncContext {
  setZoomed: (zoomed: boolean) => void;
  syncRef: MutableRefObject<uPlot.SyncPubSub>;
  zoomed: boolean;
}

interface SyncableBounds {
  boundsOptions: Partial<uPlot.Options>;
  setZoomed: (zoomed: boolean) => void;
  zoomed: boolean;
}

interface Props {
  children: React.ReactNode;
}

const SyncContext = createContext<SyncContext | undefined>(undefined);

export const SyncProvider: React.FC<Props> = ({ children }) => {
  const syncRef = useRef(uPlot.sync('x'));
  const [zoomed, setZoomed] = useState(false);

  useEffect(() => {
    if (!zoomed) {
      syncRef.current.plots.forEach((chart: uPlot) => {
        chart.setData(chart.data, true);
      });
    }
  }, [zoomed]);

  return (
    <SyncContext.Provider value={{ setZoomed, syncRef, zoomed }}>{children}</SyncContext.Provider>
  );
};

export const useSyncableBounds = (): SyncableBounds => {
  const [zoomed, setZoomed] = useState(false);
  const mouseX = useRef<number | undefined>(undefined);
  const syncContext = useContext(SyncContext);
  const zoomSetter = syncContext?.setZoomed ?? setZoomed;
  const syncRef: MutableRefObject<uPlot.SyncPubSub> | undefined = syncContext?.syncRef;

  const boundsOptions = useMemo(() => {
    return {
      cursor: {
        bind: {
          dblclick: (chart: uPlot, _target: EventTarget, handler: (e: MouseEvent) => null) => {
            return (e: MouseEvent) => {
              zoomSetter(false);
              return handler(e);
            };
          },
          mousedown: (_uPlot: uPlot, _target: EventTarget, handler: (e: MouseEvent) => null) => {
            return (e: MouseEvent) => {
              mouseX.current = e.clientX;
              return handler(e);
            };
          },
          mouseup: (_uPlot: uPlot, _target: EventTarget, handler: (e: MouseEvent) => null) => {
            return (e: MouseEvent) => {
              if (mouseX.current != null && Math.abs(e.clientX - mouseX.current) > 5) {
                zoomSetter(true);
              }
              mouseX.current = undefined;
              return handler(e);
            };
          },
        },
        drag: syncRef ? { dist: 5, uni: 10, x: true } : { dist: 5, uni: 10, x: true, y: true },
        sync: syncRef && {
          key: syncRef.current.key,
          scales: [syncRef.current.key, null],
          setSeries: false,
        },
      },
    };
  }, [zoomSetter, syncRef]) as Partial<uPlot.Options>;

  return syncContext ? { ...syncContext, boundsOptions } : { boundsOptions, setZoomed, zoomed };
};
