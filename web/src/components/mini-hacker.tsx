"use client";

import { useEffect, useState } from "react";

import { Eye, EyeOff, Pause, Play, Terminal } from "lucide-react";

import styles from "./mini-hacker.module.css";

const STORAGE_KEY = "tyan_mini_hacker";

export function MiniHacker() {
  const [ready, setReady] = useState(false);
  const [visible, setVisible] = useState(true);
  const [paused, setPaused] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(false);

  useEffect(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null");
      if (typeof saved?.visible === "boolean") setVisible(saved.visible);
      if (typeof saved?.paused === "boolean") setPaused(saved.paused);
    } catch {
      /* Storage can be unavailable; the mascot remains optional. */
    }
    const media = window.matchMedia("(prefers-reduced-motion: reduce)");
    setReducedMotion(media.matches);
    const update = () => setReducedMotion(media.matches);
    media.addEventListener("change", update);
    setReady(true);
    return () => media.removeEventListener("change", update);
  }, []);

  function updatePreferences(nextVisible: boolean, nextPaused: boolean) {
    setVisible(nextVisible);
    setPaused(nextPaused);
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ visible: nextVisible, paused: nextPaused }));
    } catch {
      /* Controls still work when preferences cannot be saved. */
    }
  }

  if (!ready) return null;

  const pauseLabel = paused ? "Resume mini hacker" : "Pause mini hacker";
  const pauseTitle = paused ? "Resume" : "Pause";

  return (
    <div className={styles.layer} data-paused={paused || reducedMotion} data-testid="mini-hacker">
      {visible && (
        <div className={styles.runner} aria-hidden="true">
          <div className={styles.character}>
            <svg viewBox="0 0 80 90" fill="none" className={styles.sprite}>
              <title>TYAN mini hacker cat</title>
              <ellipse cx="40" cy="84" rx="23" ry="4" fill="#000" opacity=".2" />
              <g className={styles.leftFoot}>
                <path d="M26 68h12v11a4 4 0 0 1-4 4H23v-6h3z" fill="#183e35" stroke="#73efbd" strokeWidth="1.5" />
              </g>
              <g className={styles.rightFoot}>
                <path d="M42 68h12v9h3v6H46a4 4 0 0 1-4-4z" fill="#183e35" stroke="#73efbd" strokeWidth="1.5" />
              </g>
              <g className={styles.bob}>
                <path d="M21 53q19-13 38 0l5 21H16z" fill="#102923" stroke="#73efbd" strokeWidth="1.5" />
                <path
                  d="m17 28 1-19 16 9h12l16-9 1 19q7 6 7 17c0 17-60 17-60 0q0-11 7-17Z"
                  fill="#102923"
                  stroke="#73efbd"
                  strokeWidth="2"
                  strokeLinejoin="round"
                />
                <path d="m22 18 8 5-9 6zM58 18l-8 5 9 6z" fill="#2b6c56" />
                <rect x="18" y="29" width="44" height="25" rx="12" fill="#061510" />
                <g className={styles.eyes}>
                  <path d="M27 39q3-5 6 0M47 39q3-5 6 0" stroke="#8effcd" strokeWidth="3" strokeLinecap="round" />
                </g>
                <path d="M36 45q4 4 8 0" stroke="#8effcd" strokeWidth="1.7" strokeLinecap="round" />
                <ellipse cx="24" cy="44" rx="3" ry="1.5" fill="#f494b6" opacity=".6" />
                <ellipse cx="56" cy="44" rx="3" ry="1.5" fill="#f494b6" opacity=".6" />
                <path d="M27 56v7M53 56v7" stroke="#73efbd" strokeWidth="1.5" strokeLinecap="round" />
                <rect x="21" y="59" width="38" height="20" rx="3" fill="#132c26" stroke="#73efbd" strokeWidth="1.5" />
                <path
                  d="m29 66 4 3-4 3m9 0h7"
                  stroke="#8effcd"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path d="M18 79h44" stroke="#73efbd" strokeWidth="2.5" strokeLinecap="round" />
                <circle cx="66" cy="25" r="2" fill="#8effcd" />
                <path d="M9 18v6m-3-3h6" stroke="#8effcd" strokeWidth="1.5" strokeLinecap="round" />
              </g>
            </svg>
          </div>
        </div>
      )}
      <fieldset className={styles.controls} aria-label="Mini hacker mascot controls">
        <Terminal size={12} aria-hidden="true" />
        <span className={styles.label}>MINI HACKER</span>
        {visible && (
          <button
            type="button"
            disabled={reducedMotion}
            onClick={() => updatePreferences(visible, !paused)}
            aria-label={reducedMotion ? "Motion disabled by system preference" : pauseLabel}
            title={reducedMotion ? "Respects your reduced-motion preference" : pauseTitle}
          >
            {paused || reducedMotion ? <Play size={12} /> : <Pause size={12} />}
          </button>
        )}
        <button
          type="button"
          onClick={() => updatePreferences(!visible, paused)}
          aria-label={visible ? "Hide mini hacker" : "Show mini hacker"}
          title={visible ? "Hide mascot" : "Show mascot"}
        >
          {visible ? <EyeOff size={12} /> : <Eye size={12} />}
        </button>
      </fieldset>
    </div>
  );
}
