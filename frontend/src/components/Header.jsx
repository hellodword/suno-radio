import React from "react";
import styles from "./Header.module.css";

const Header = () => {
  return (
    <div className={styles.wrapper}>
      <h1 className={styles.title}>Suno Radio</h1>
      <p className={styles.description}>
        📻🤖
      </p>
    </div>
  );
};

export default Header;
