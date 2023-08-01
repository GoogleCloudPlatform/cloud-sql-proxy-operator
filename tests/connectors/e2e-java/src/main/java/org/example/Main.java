package org.example;

import com.google.api.client.util.DateTime;
import com.zaxxer.hikari.HikariConfig;
import com.zaxxer.hikari.HikariDataSource;
import java.io.OutputStream;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.Instant;
import java.time.format.DateTimeFormatter;
import java.util.Properties;
import java.util.logging.ConsoleHandler;
import java.util.logging.Level;
import java.util.logging.Logger;
import java.util.logging.SimpleFormatter;

/*

      DB_NAME=db
      DB_USER=hessjc-csql-operator-02:us-central1:mysql2d971003318a3077402fhessjc
      DB_PASS=604a0cc12f342b9ae9f9
      DB_USER=dbuser

      ;E2E_CONNECT_PATTERN=short;E2E_DB_TYPE=mysql

 */
public class Main {
  private static final Logger logger;

  static {
    ConsoleHandler handler = new ConsoleHandler(){{
      setOutputStream(System.out);
      setFormatter(new JsonFormatter());
      setLevel(Level.ALL);
    }};



    Logger root = Logger.getLogger("");
    root.addHandler(handler);
    root.setLevel(Level.INFO);
    logger = Logger.getLogger(Main.class.getName());
    logger.setLevel(Level.ALL);
    Logger.getLogger("com.google.cloud.sql.core").setLevel(Level.ALL);
    // bypass root logger to log fine messages
    // Logger l = Logger.getLogger("com.google.cloud.sql.core.CloudSqlInstance");
    // l.setLevel(Level.ALL);
    // l.setUseParentHandlers(false);
    // l.addHandler(handler);
  }

  public static void main(String[] args) {
    String ipType = System.getenv("E2E_IP_TYPES");
    String style = System.getenv("E2E_CONNECT_PATTERN");
    String dbType = System.getenv("E2E_DB_TYPE");
    boolean invalid = false;
    if (style == null || style.isEmpty() || !( style.equals("short") || style.equals("long"))) {
      logger.info("Set E2E_CONNECT_PATTERN to `short` or `long`");
      invalid = true;
    }
    if (dbType == null || dbType.isEmpty() || !( dbType.equals("mysql") || dbType.equals("postgres")  || dbType.equals("sqlserver"))) {
      logger.info("Set E2E_DB_TYPE to `mysql` or `postgres` or `sqlserver`");
      invalid = true;
    }
    if(invalid) {
      System.exit(1);
    }
    String dbUser = System.getenv("DB_USER");
    String dbPass = System.getenv("DB_PASS");
    String dbName = System.getenv("DB_NAME");
    String dbName1 = System.getenv("DB_NAME_1");
    String dbInstance = System.getenv("DB_INSTANCE");
    String dbInstance1 = System.getenv("DB_INSTANCE_1");

    HikariDataSource connectionPool = makeConnectionPool(dbType, ipType, dbInstance, dbName, dbUser, dbPass);
    HikariDataSource connectionPool1;
    if ( dbInstance1 != null && ! dbInstance1.isEmpty()) {
      connectionPool1 = makeConnectionPool(dbType, ipType, dbInstance1, dbName1, dbUser, dbPass);
    } else {
      connectionPool1 = makeConnectionPool(dbType, ipType, dbInstance, dbName1, dbUser, dbPass);
    }

    Thread t0 = new Thread(()->{
      try {
        runConnections(connectionPool, style);
      } catch (Throwable e) {
        logger.log(Level.WARNING, "Caught exception while connecting to "+connectionPool.getJdbcUrl(), e);
        System.exit(1);
      }
    });

    Thread t1 = new Thread(()->{
      try {
        runConnections(connectionPool1, style);
      } catch (Exception e) {
        logger.log(Level.WARNING, "Caught exception while connecting to "+connectionPool.getJdbcUrl(), e);
        System.exit(1);
      }
    });

    t0.start();
    t1.start();
  }

  public static void runConnections(HikariDataSource connectionPool, String style) {
    if ("long".equals(style)) {
      hikariLongConnections(connectionPool);
    }
    if ("short".equals(style)) {
      hikariShortConnections(connectionPool);
    }
  }

  public static void hikariLongConnections(HikariDataSource connectionPool) {
    Connection conn = null;
    try {
      conn = connectionPool.getConnection();
    } catch (SQLException e) {
      throw new RuntimeException(e);
    }
    while (true) {
      try {
        ResultSet rs = conn.createStatement().executeQuery("select 1");
        while (rs.next()) {
          int v = rs.getInt(1);
          if( v == 1) {
            logger.fine("Success: " + DateTimeFormatter.ISO_INSTANT.format(Instant.now()));
          }
        }
      } catch (SQLException e) {
        logger.log(Level.WARNING,"Error: " + DateTimeFormatter.ISO_INSTANT.format(Instant.now()), e);
      }
      try {
        Thread.sleep(60L * 1000L);
      } catch (InterruptedException e) {
        logger.info("Interrupted. Exiting.");
        break;
      }
    }

  }

  public static void hikariShortConnections(HikariDataSource connectionPool) {
    while (true) {
      try (Connection conn = connectionPool.getConnection()) {
        ResultSet rs = conn.createStatement().executeQuery("select 1");
        while (rs.next()) {
          int v = rs.getInt(1);
          if( v == 1) {
            logger.fine("Success: " + DateTimeFormatter.ISO_INSTANT.format(Instant.now()));
          }
        }
      } catch (SQLException e) {
        logger.log(Level.WARNING,"Error: " + DateTimeFormatter.ISO_INSTANT.format(Instant.now()), e);
      }
      try {
        Thread.sleep(60L * 1000L);
      } catch (InterruptedException e) {
        logger.fine("Interrupted. Exiting.");
        break;
      }
    }

  }

  private static HikariDataSource makeConnectionPool(String dbType, String ipType, String instance, String dbName, String dbUser,String dbPass) {

    // Initialize connection pool
    HikariConfig config = new HikariConfig();
    config.setConnectionTimeout(10000); // 10s
    config.setMinimumIdle(0); // no idle connections
    config.setMaxLifetime(10000); //10s
    config.setUsername(dbUser);
    config.setPassword(dbPass);

    if("mysql".equals(dbType)) {
      config.setJdbcUrl(String.format("jdbc:mysql:///%s", dbName));
      config.addDataSourceProperty("cloudSqlInstance", instance);
      config.addDataSourceProperty("socketFactory", "com.google.cloud.sql.mysql.SocketFactory");
      if(ipType != null && ! ipType.isEmpty()) {
        config.addDataSourceProperty("ipTypes", ipType);
      }
    }
    else if("postgres".equals(dbType)) {
      config.setJdbcUrl(String.format("jdbc:postgresql:///%s", dbName));
      config.addDataSourceProperty("cloudSqlInstance", instance);
      config.addDataSourceProperty("sslmode", "disable");
      config.addDataSourceProperty("socketFactory", "com.google.cloud.sql.postgres.SocketFactory");
      if(ipType != null && ! ipType.isEmpty()) {
        config.addDataSourceProperty("ipTypes", ipType);
      }
    }
    else if("sqlserver".equals(dbType)) {
      if(ipType != null && ! ipType.isEmpty()) {
        config.setJdbcUrl(String.format("jdbc:sqlserver://localhost;databaseName=%s&ipTypes=%s", dbName,
            ipType));
      } else {
        config.setJdbcUrl(String.format("jdbc:sqlserver://localhost;databaseName=%s", dbName));
      }

      config.setDataSourceClassName("com.microsoft.sqlserver.jdbc.SQLServerDataSource");
      config.addDataSourceProperty("socketFactoryClass", "com.google.cloud.sql.sqlserver.SocketFactory");
      config.addDataSourceProperty("socketFactoryConstructorArg", instance);
      config.addDataSourceProperty("encrypt", "false");
      config.setConnectionTimeout(10000); // 10s
    }

    HikariDataSource connectionPool = new HikariDataSource(config);
    return connectionPool;
  }
}