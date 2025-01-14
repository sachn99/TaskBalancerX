use actix_web::{post, web, App, HttpServer, Responder};
use serde::{Deserialize};
use std::sync::Arc;
use tokio::task;

#[derive(Deserialize)]
struct Task {
    file: String,
}

#[post("/process")]
async fn process_task(task: web::Json<Task>) -> impl Responder {

    if task.file.is_empty(){
        return web::HttpResponse::BadRequest()>body("File name is missing!");
    }

    println!("Processing file: {}", task.file);

    let file_name = task.file.clone();
    let task = Arc::new(task);
    // Add logic to handle the file (e.g., processing image)

    task::spawn(async move {
            println!("Processing file: {}", file_name);

            // Simulate file processing (e.g., image resizing)
            tokio::time::sleep(std::time::Duration::from_secs(2)).await;

            println!("Completed processing file: {}", file_name);
        });

        web::HttpResponse::Ok().body(format!("Processing started for file: {}", task.file))
    // Simulate processing time
    //std::thread::sleep(std::time::Duration::from_secs(2));

    //web::HttpResponse::oK().body(format!("Processed file: {}", task.file))
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
