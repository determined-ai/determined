// @ts-nocheck
import React, {
  createContext,
  MutableRefObject,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import uPlot from 'uplot';

import { distance } from 'utils/chart';

const timestampInMinutes = t => new Date(t).toTimeString().slice(3, 9);

interface SyncableBounds {
  boundsOptions: Partial<uPlot.Options>;
  isZoomed: (zoomed: boolean) => void;
  zoomed: boolean;
}

const SyncContext = createContext();

const getChartMin = (chart: UPlot) => chart.scales.x.min;
const getChartMax = (chart: UPlot) => chart.scales.x.max;

export const SyncProvider = ({ children }) => {
  const syncRef = useRef(uPlot.sync('x'));
  const [ syncedMin, setSyncedMin ] = useState();
  const [ syncedMax, setSyncedMax ] = useState();
  const [ queueSignal, setQueueSignal ] = useState(0);
  const [ zoomed, setZoomed ] = useState(false);
  const fireQueue = useCallback(
    () => setQueueSignal(prevSignal => (prevSignal + 1) % 1000)
    , [ setQueueSignal ],
  );

  const minMaxRef = useRef({});

  const updateQueue = useRef([]);

  useEffect(() => {
    // console.log('process queue', JSON.stringify(updateQueue.current));
    while (updateQueue.current.length) {
      const [ min, max ] = updateQueue.current.pop(); // most recent most likely to be real update?
      // if zoom, empty it out
      if (syncedMin == null || min < syncedMin) setSyncedMin(min);
      if (syncedMax == null || max > syncedMax) setSyncedMax(max);
    }
  }, [ queueSignal, syncedMin, syncedMax ]);

  useEffect(() => {
    // console.log('new bounds', timestampInMinutes(syncedMin), timestampInMinutes(syncedMax));
    minMaxRef.current.max = syncedMax;
    minMaxRef.current.min = syncedMin;
    if(!zoomed) {
      syncRef.current.plots.forEach((chart: uPlot) => {
        const chartMin = getChartMin(chart);
        const chartMax = getChartMax(chart);
        // console.log('set bounds', chart.title, syncedMin, syncedMax);
        if (chartMin > syncedMin || chartMax < syncedMax) {
          chart.setScale('x', { max: syncedMax, min: syncedMin });

        }
      });
    }
  }, [ syncedMin, syncedMax, zoomed ]);

  const dispatchScaleUpdate = useCallback(
    (chart: uPlot, scaleKey) => {
      if(scaleKey === 'x') {
        updateQueue.current.push([ getChartMin(chart), getChartMax(chart) ]);
        fireQueue();
      }
    },
    [ fireQueue ],
  );

  return (
    <SyncContext.Provider
      value={{ dispatchScaleUpdate, minMaxRef, setZoomed, syncRef, zoomed }}>
      {children}
    </SyncContext.Provider>
  );
};

export const useSyncableBounds = (): SyncableBounds => {
  const [ zoomed, setZoomed ] = useState(false);
  const mousePosition = useRef();
  const syncContext = useContext(SyncContext);
  const zoomSetter = syncContext?.setZoomed ?? setZoomed;
  const scaleUpdateDispatcher = syncContext?.dispatchScaleUpdate;
  const minMaxRef = syncContext?.minMaxRef;
  const syncRef: MutableRefObject<uPlot.SyncPubSub> = syncContext?.syncRef;

  const boundsOptions = useMemo(() => ({
    cursor: {
      bind: {
        dblclick: (chart: uPlot, _target: EventTarget, handler: (e: Event) => void) => {
          return (e: Event) => {
            zoomSetter(false);
            if (minMaxRef){
              chart.setScale('x', { max: minMaxRef.current.max, min: minMaxRef.current.max });
            } else {
              handler(e);
            }
          };
        },
        mousedown: (_uPlot: uPlot, _target: EventTarget, handler: (e: Event) => void) => {
          return (e: MouseEvent) => {
            mousePosition.current = [ e.clientX, e.clientY ];
            handler(e);
          };
        },
        mouseup: (_uPlot: uPlot, _target: EventTarget, handler: (e: Event) => void) => {
          return (e: MouseEvent) => {
            if (!mousePosition.current) {
              handler(e);
              return;
            }
            if (distance(
              e.clientX,
              e.clientY,
              mousePosition.current[0],
              mousePosition.current[1],
            ) > 5) {
              zoomSetter(true);
            }
            mousePosition.current = undefined;
            handler(e);
          };
        },

      },
      drag: { dist: 5, uni: 10, x: true, y: true },
      sync: syncRef && {
        key: syncRef.current.key,
        scales: [ syncRef.current.key, null ],
        setSeries: false,
      },
    },
    hooks: scaleUpdateDispatcher && { setScale: [ scaleUpdateDispatcher ] },
  }), [ zoomSetter, scaleUpdateDispatcher, syncRef, minMaxRef ]);

  return syncContext ? { ...syncContext, boundsOptions } : { boundsOptions, zoomed };
};

// use sync: use  sync context else setState
