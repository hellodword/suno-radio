import React from "react";
import styles from "./Footer.module.css";

const Footer = () => {
  return (
    <div className={styles.wrapper}>
      <p className={styles.contribute}>
        Powered by{" "}
        <a
          className="link"
          target="_blank"
          rel="noopener noreferrer"
          href="https://suno.com/"
        >
          Suno AI{" "}
        </a>
        and{" "}
        <a
          className="link"
          target="_blank"
          rel="noopener noreferrer"
          href="https://github.com/madzadev/audio-player"
        >
          audio-player{" "}
        </a>
      </p>
    </div>
  );
};

export default Footer;
