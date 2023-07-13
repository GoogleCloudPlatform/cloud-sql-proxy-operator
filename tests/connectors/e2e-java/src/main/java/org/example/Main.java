package org.example;

import com.google.api.client.util.DateTime;
import com.zaxxer.hikari.HikariConfig;
import com.zaxxer.hikari.HikariDataSource;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.Instant;
import java.time.format.DateTimeFormatter;
import java.util.Properties;

/*

      DB_NAME=db
      DB_USER=hessjc-csql-operator-02:us-central1:mysql2d971003318a3077402fhessjc
      DB_PASS=604a0cc12f342b9ae9f9
      DB_USER=dbuser

 */
public class Main {

  public static void main(String[] args) {
    String ipType = System.getenv("E2E_IP_TYPES");
    String style = System.getenv("E2E_CONNECT_PATTERN");
    String dbType = System.getenv("E2E_DB_TYPE");
    boolean invalid = false;
    if (style == null || style.isEmpty() || !( style.equals("short") || style.equals("long"))) {
      System.err.println("Set E2E_CONNECT_PATTERN to `short` or `long`");
      invalid = true;
    }
    if (dbType == null || dbType.isEmpty() || !( dbType.equals("mysql") || dbType.equals("postgres")  || dbType.equals("sqlserver"))) {
      System.err.println("Set E2E_DB_TYPE to `mysql` or `postgres` or `sqlserver`");
      invalid = true;
    }
    if(invalid) {
      System.exit(1);
    }

    HikariDataSource connectionPool = makeConnectionPool(dbType, ipType);
    runConnections(connectionPool, style);
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
            System.out.println("Success: " + DateTimeFormatter.ISO_INSTANT.format(Instant.now()));
          }
        }
      } catch (SQLException e) {
        System.out.println();
        System.out.println("Error: " + DateTimeFormatter.ISO_INSTANT.format(Instant.now()));
        e.printStackTrace(System.out);
        System.out.println();
      }
      try {
        Thread.sleep(60L * 1000L);
      } catch (InterruptedException e) {
        System.out.println("Interrupted. Exiting.");
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
            System.out.println("Success: " + DateTimeFormatter.ISO_INSTANT.format(Instant.now()));
          }
        }
      } catch (SQLException e) {
        System.out.println();
        System.out.println("Error: " + DateTimeFormatter.ISO_INSTANT.format(Instant.now()));
        e.printStackTrace(System.out);
        System.out.println();
      }
      try {
        Thread.sleep(60L * 1000L);
      } catch (InterruptedException e) {
        System.out.println("Interrupted. Exiting.");
        break;
      }
    }

  }

  private static HikariDataSource makeConnectionPool(String dbType, String ipType) {

    // Initialize connection pool
    HikariConfig config = new HikariConfig();
    config.setConnectionTimeout(10000); // 10s
    config.setMinimumIdle(0);
    config.setUsername(System.getenv("DB_USER"));
    config.setPassword(System.getenv("DB_PASS"));

    if("mysql".equals(dbType)) {
      config.setJdbcUrl(String.format("jdbc:mysql:///%s", System.getenv("DB_NAME")));
      config.addDataSourceProperty("cloudSqlInstance", System.getenv("DB_INSTANCE"));
      config.addDataSourceProperty("socketFactory", "com.google.cloud.sql.mysql.SocketFactory");
      if(ipType != null && ! ipType.isEmpty()) {
        config.addDataSourceProperty("ipTypes", ipType);
      }
    }
    else if("postgres".equals(dbType)) {
      config.setJdbcUrl(String.format("jdbc:postgresql:///%s", System.getenv("DB_NAME")));
      config.addDataSourceProperty("cloudSqlInstance", System.getenv("DB_INSTANCE"));
      config.addDataSourceProperty("sslmode", "disable");
      config.addDataSourceProperty("socketFactory", "com.google.cloud.sql.postgres.SocketFactory");
      if(ipType != null && ! ipType.isEmpty()) {
        config.addDataSourceProperty("ipTypes", ipType);
      }
    }
    else if("sqlserver".equals(dbType)) {
      if(ipType != null && ! ipType.isEmpty()) {
        config.setJdbcUrl(String.format("jdbc:sqlserver://localhost;databaseName=%s&ipTypes=%s", System.getenv("DB_NAME"),
            ipType));
      } else {
        config.setJdbcUrl(String.format("jdbc:sqlserver://localhost;databaseName=%s", System.getenv("DB_NAME")));
      }

      config.setDataSourceClassName("com.microsoft.sqlserver.jdbc.SQLServerDataSource");
      config.addDataSourceProperty("socketFactoryClass", "com.google.cloud.sql.sqlserver.SocketFactory");
      config.addDataSourceProperty("socketFactoryConstructorArg", System.getenv("DB_INSTANCE"));
      config.addDataSourceProperty("encrypt", "false");
      config.setConnectionTimeout(10000); // 10s
    }

    HikariDataSource connectionPool = new HikariDataSource(config);
    return connectionPool;
  }
}