$(document).ready(function() {
    var canvas = new fabric.Canvas('canvas');
    canvas.isDrawingMode = 1;
    canvas.freeDrawingBrush.color = "blue";
    canvas.freeDrawingBrush.width = 5;
    canvas.renderAll();

    var chartCtx = $("#chart-canvas")[0].getContext('2d')
    var chart = new Chart(chartCtx, {
        // The type of chart we want to create
        type: 'bar',
    
        // The data for our dataset
        data: {
            labels: ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9'],
            datasets: [{
                label: 'Prediction',
                backgroundColor: 'rgb(255, 99, 132)',
                borderColor: 'rgb(255, 99, 132)',
                data: [0, 10, 5, 2, 20, 30, 45]
            }]
        },
    
        // Configuration options go here
        options: {
            maintainAspectRatio: false
        }
    });

    $("#clear-btn").click(function() {
        canvas.clear();
    });

    $('#modal-deploy-model').on('show.bs.modal', function (event) {
        var button = $(event.relatedTarget) // Button that triggered the modal
        var kind = button.data('kind') // Extract info from data-* attributes
        $("#is-new-model").val(kind)
    })

    $("#modal-deploy-ok").click(function() {
        // send ajax
        var isNewModel = $("#is-new-model").val() == "new"
        $.ajax({
            url: '/model:deploy',
            type: "POST",
            dataType: 'text',
            contentType: "application/json; charset=utf-8",
            data: JSON.stringify(
                {
                    "model-base-dir": $("#model-base-dir").val(),
                    "model-name": $("#model-name").val(),
                    "num-replicas": parseInt($("#num-replicas").val()),
                    "is-new-model": isNewModel
                }
            ),
            success : function(result) {
                $("#modal-deploy-model").modal("hide")
                
                // change statuses
                if (isNewModel) {
                    $("#deploy-new-model-btn").removeClass("btn-primary").addClass("btn-info")
                    $("#new-model-only-btn").prop("disabled", false)
                    $("#deploy-new-model-btn").text("Re-Deploy")
                } else {
                    $("#deploy-cur-model-btn").removeClass("btn-primary").addClass("btn-info")
                    $("#cur-model-only-btn").prop("disabled", false)
                    $("#deploy-cur-model-btn").text("Re-Deploy")
                }

                var curModelDisabled = $("#cur-model-only-btn").prop("disabled")
                var newModelDisabled = $("#new-model-only-btn").prop("disabled")
                if (!curModelDisabled && !newModelDisabled) {
                    $("#canary-btn").prop("disabled", false)
                    $("#canary-range").prop("disabled", false)
                    $("#canary-weight-text").prop("hidden", false)
                }

                $("#set-strategy-btn").prop("disabled", false)
                $("#predict-btn").prop("disabled", false)
            },
            error: function(xhr, resp, text) {
                console.log(xhr, resp, text);
            }
        })
    })

    $("#set-strategy-btn").click(function() {
        // set strategy
        var strategyBtns = $("input:radio[name='predict-radio-options']");
        var strategyIdx = strategyBtns.index(strategyBtns.filter(':checked'))
        $.ajax({
            url: '/model/strategy',
            type: "PUT",
            dataType: 'text',
            contentType: "application/json; charset=utf-8",
            data: JSON.stringify(
                {
                    "strategy": strategyIdx,
                    "weight": parseInt($("#canary-range").val())
                }
            ),
            success : function(result) {
                switch (strategyIdx) {
                    case 0:
                        $("#cur-strategy-text").text("Strategy - Current Model Only")
                        break;
                    case 1:
                        $("#cur-strategy-text").text("Strategy - New Model Only")
                        break;
                    case 2:
                        $("#cur-strategy-text").text("Strategy - Canary")
                        break;
                    default:
                        console.log("Unknown strategy")
                }
            },
            error: function(xhr, resp, text) {
                console.log(xhr, resp, text);
            }
        })
    })

    $("#predict-btn").click(function() {
        var hiddenCanvasCtx = $("#hidden-resized-canvas")[0].getContext('2d');
        hiddenCanvasCtx.clearRect(0, 0, 28, 28)
        hiddenCanvasCtx.drawImage($("#canvas")[0], 0, 0, 28, 28);
        var imageData = hiddenCanvasCtx.getImageData(0, 0, 28, 28)
        var pixelData = imageData.data
        var greyScaledPixelData = []
        for (var i=0; i<pixelData.length; i++) {
            // only use B in RGBA
            if (i % 4 == 2) {
                // MNIST data has black background
                greyScaledPixelData.push((pixelData[i] / 255.0))
            }
        }

        $.ajax({
            url: '/model:predict',
            type: "POST",
            dataType: 'json',
            contentType: "application/json; charset=utf-8",
            data: JSON.stringify(
                greyScaledPixelData
            ),
            success : function(result) {
                chart.data.datasets.forEach((dataset) => {
                    dataset.data.pop();
                });

                chart.data.datasets[0].data = result
                chart.update();
            },
            error: function(xhr, resp, text) {
                console.log(xhr, resp, text);
            }
        })
    })

})