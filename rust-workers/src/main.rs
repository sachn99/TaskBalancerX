use actix_web::{post, web, App, HttpServer, Responder};
use serde::{Deserialize};

#[derive(Deserialize)]
struct Task {
    file: String,
}

#[post("/process")]
async fn process_task(task: web::Json<Task>) -> impl Responder {
    println!("Processing file: {}", task.file);
    // Add logic to handle the file (e.g., processing image)

    // Simulate processing time
    std::thread::sleep(std::time::Duration::from_secs(2));

    format!("Processed file: {}", task.file)
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    HttpServer::new(|| {
        App::new().service(process_task)
    })
    .bind("127.0.0.1:8081")?
    .run()
    .await
}
